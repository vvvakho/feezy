package workflows

import (
	"errors"
	"fmt"
	"time"

	"github.com/vvvakho/feezy/billing/service/domain"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type AddItemSignal struct {
	LineItem domain.Item
}

type RemoveItemSignal struct {
	LineItem domain.Item
}

type CloseBillSignal struct {
	Route     string
	RequestID string
}

type CloseWorkflowSignal struct {
	Route     string
	RequestID string
}

type SignalRoute struct {
	Name string
}

var AddLineItemRoute = SignalRoute{
	Name: "addLineItem",
}

var RemoveLineItemRoute = SignalRoute{
	Name: "removeLineItem",
}

var CloseBillRoute = SignalRoute{
	Name: "CloseBillSignal",
}

var CloseWorkflowRoute = SignalRoute{
	Name: "CloseWorkflowSignal",
}

// Register Temporal signal handlers for processing bill events.
func registerSignalHandlers(
	ctx workflow.Context,
	mu workflow.Mutex,
	selector workflow.Selector,
	addLineItemChan,
	removeLineItemChan,
	closeBillChan,
	closeWorkflowChan workflow.ReceiveChannel,
	bill *domain.Bill,
	logger log.Logger,
) {

	// Register a handler for adding line item to bill
	selector.AddReceive(addLineItemChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := HandleAddLineItemSignal(ctx, mu, c, bill); err != nil {
			logger.Error("Adding item to bill", "Error", err)
		}
	})

	// Register a handler for removing line item from bill
	selector.AddReceive(removeLineItemChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := HandleRemoveLineItemSignal(ctx, mu, c, bill); err != nil {
			logger.Error("Removing item from bill", "Error", err)
		}
	})

	// Register a handler for closing bill (through a signal)
	selector.AddReceive(closeBillChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := HandleCloseBillSignal(ctx, mu, c, bill, logger); err != nil {
			logger.Error("Closing bill", "Error", err)
		}
	})

	// Register a handler for closing workflow
	selector.AddReceive(closeWorkflowChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := HandleCloseWorkflowSignal(ctx, mu, c, bill, logger); err != nil {
			logger.Error("Closing workflow", "Error", err)
		}
	})
}

// Handler function for adding line item to bill.
func HandleAddLineItemSignal(ctx workflow.Context, mu workflow.Mutex, c workflow.ReceiveChannel, bill *domain.Bill) error {
	// Use mutex locking for safe concurrency
	err := mu.Lock(ctx)
	if err != nil {
		return fmt.Errorf("Error locking mutex: %v", err)
	}
	defer mu.Unlock()

	if bill.Status != domain.BillOpen {
		return fmt.Errorf("Bill is no longer open")
	}

	var addSignal AddItemSignal
	c.Receive(ctx, &addSignal)

	lineItem := addSignal.LineItem
	if err := bill.AddLineItem(lineItem); err != nil {
		return err
	}

	if err := bill.CalculateTotal(); err != nil {
		return err
	}

	bill.UpdatedAt = time.Now()

	return nil
}

// Handler function for removing line item from bill.
func HandleRemoveLineItemSignal(ctx workflow.Context, mu workflow.Mutex, c workflow.ReceiveChannel, bill *domain.Bill) error {
	// Use mutex locking for safe concurrency
	err := mu.Lock(ctx)
	if err != nil {
		return fmt.Errorf("Error locking mutex: %v", err)
	}
	defer mu.Unlock()

	if bill.Status != domain.BillOpen {
		return fmt.Errorf("Bill is no longer open")
	}

	var removeSignal RemoveItemSignal
	c.Receive(ctx, &removeSignal)

	lineItem := removeSignal.LineItem
	if err := bill.RemoveLineItem(lineItem); err != nil {
		return err
	}

	bill.UpdatedAt = time.Now()

	return nil
}

// Handler function for closing bill through a signal call.
func HandleCloseBillSignal(ctx workflow.Context, mu workflow.Mutex, c workflow.ReceiveChannel, bill *domain.Bill, logger log.Logger) error {
	for {
		// If the bill is already closed, ignore further signals
		if bill.Status == domain.BillClosed {
			logger.Warn("Received close bill signal, but bill is already closed", "BillID", bill.ID)
			return fmt.Errorf("Bill already closed")
		}

		// Use mutex locking for safe concurrency
		err := mu.Lock(ctx)
		if err != nil {
			return fmt.Errorf("Error locking mutex: %v", err)
		}
		defer mu.Unlock()

		// Inititate bill closing status
		bill.Status = domain.BillClosing
		bill.UpdatedAt = time.Now()

		// Unpack the signal contents
		var closeSignal CloseBillSignal
		c.Receive(ctx, &closeSignal)

		// Calculate bill total or throw an error in case of failure
		if err := bill.CalculateTotal(); err != nil {
			logger.Error("Error calculating bill total", "Error", err)
			bill.Status = domain.BillOpen
			return fmt.Errorf("Error closing bill: %v", err)
		}

		// Set retry policy for transient failures (e.g., network issues)
		retryPolicy := &temporal.RetryPolicy{
			InitialInterval:    time.Second * 2,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    5,
		}
		activityOptions := workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			RetryPolicy:         retryPolicy,
		}

		ctx = workflow.WithActivityOptions(ctx, activityOptions)

		// Initiate the activity to move the bill to a closed_bills database table
		err = workflow.ExecuteActivity(ctx, AddClosedBillToDB, bill, closeSignal.RequestID).Get(ctx, nil)
		if err != nil {
			var appErr *temporal.ApplicationError
			// Check the type of error to determine action
			if errors.As(err, &appErr) {
				if appErr.Type() == "DuplicateRequestError" {
					// Ignore request and exit if it is a duplicate request
					logger.Warn("Duplicate close request detected, ignoring", "RequestID", closeSignal.RequestID)
					return fmt.Errorf("duplicate close request ignored")
				} else if appErr.Type() == "UserInputError" {
					// Cancel request if error is due to user input
					// Set the bill status back to open
					logger.Error("Invalid input, rejecting close request", "Error", appErr)
					bill.Status = domain.BillOpen
					return err
				} else if appErr.Type() == "InvalidRequestError" {
					// Cancel request if error is due to invalid request
					// Set the bill status back to open
					logger.Error("Invalid request, rejecting close request", "Error", appErr)
					bill.Status = domain.BillOpen
					return err
				}
			}
			// If error is still present after the retry policy and is not of the above type:
			logger.Error("Error executing AddClosedBillToDB activity", "Error", err)
			return err
		}

		// Successfully closed the bill, exit loop
		bill.Status = domain.BillClosed

		logger.Info("Bill successfully saved as closed in DB", "BillID", bill.ID)
		break
	}

	return nil
}

// Handler function for closing bill through an update call.
func HandleCloseBillUpdate(ctx workflow.Context, mu workflow.Mutex, bill *domain.Bill, logger log.Logger) error {
	// Set up a handler function to process CloseBillUpdate events
	err := workflow.SetUpdateHandler(ctx, "CloseBillUpdate", func(ctx workflow.Context, requestID string) (*domain.Bill, error) {
		// Check that bill is not already closed
		if bill.Status == domain.BillClosed {
			logger.Warn("Received close bill update, but bill is already closed", "BillID", bill.ID)
			return nil, fmt.Errorf("Bill already closed")
			// Check that bill is not in the middle of closing
		} else if bill.Status == domain.BillClosing {
			logger.Warn("Received close bill update, but bill is currently closing", "BillID", bill.ID)
			return nil, fmt.Errorf("Bill is in the middle of closing")
		}

		// Use mutex locking for safe concurrency
		err := mu.Lock(ctx)
		if err != nil {
			return nil, fmt.Errorf("Error locking mutex: %v", err)
		}
		defer mu.Unlock()

		// Inititate bill closing status
		bill.Status = domain.BillClosing
		bill.UpdatedAt = time.Now()

		// Calculate bill total or throw an error in case of failure
		if err := bill.CalculateTotal(); err != nil {
			logger.Error("Error calculating bill total", "Error", err)
			bill.Status = domain.BillOpen
			return nil, fmt.Errorf("Error closing bill: %v", err)
		}

		// Set retry policy for transient failures (e.g., network issues)
		retryPolicy := &temporal.RetryPolicy{
			InitialInterval:    time.Second * 2,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    5,
		}
		activityOptions := workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			RetryPolicy:         retryPolicy,
		}

		ctx = workflow.WithActivityOptions(ctx, activityOptions)

		// Initiate the activity to move the bill to a closed_bills database table
		err = workflow.ExecuteActivity(ctx, AddClosedBillToDB, bill, requestID).Get(ctx, nil)
		if err != nil {
			var appErr *temporal.ApplicationError
			// Check the type of error to determine action
			if errors.As(err, &appErr) {
				if appErr.Type() == "DuplicateRequestError" {
					// Ignore request and exit if it is a duplicate request
					logger.Warn("Duplicate close request detected, ignoring", "RequestID", requestID)
					return nil, fmt.Errorf("duplicate close request ignored")
				} else if appErr.Type() == "UserInputError" {
					// Cancel request if error is due to user input
					// Set the bill status back to open
					logger.Error("Invalid input, rejecting close request", "Error", appErr)
					bill.Status = domain.BillOpen
					return nil, err
				} else if appErr.Type() == "InvalidRequestError" {
					// Cancel request if error is due to invalid request
					// Set the bill status back to open
					logger.Error("Invalid request, rejecting close request", "Error", appErr)
					bill.Status = domain.BillOpen
					return nil, err
				}
			}
			// If error is still present after the retry policy and is not of the above type:
			logger.Error("Error executing AddClosedBillToDB activity", "Error", err)
			return nil, err
		}

		// Finish the action of closing bill
		bill.Status = domain.BillClosed
		logger.Info("Bill successfully saved as closed in DB", "BillID", bill.ID)
		return bill, nil
	})

	return err
}

// Handler function for closing a workflow through signal call.
func HandleCloseWorkflowSignal(ctx workflow.Context, mu workflow.Mutex, c workflow.ReceiveChannel, bill *domain.Bill, logger log.Logger) error {
	// Unpack the signal contents
	var closeWFSignal CloseWorkflowSignal
	c.Receive(ctx, &closeWFSignal)

	// Mutex locking for safe concurrency
	err := mu.Lock(ctx)
	if err != nil {
		return fmt.Errorf("Error locking mutex: %v", err)
	}
	defer mu.Unlock()

	// Change bill status to closed to finish workflow
	logger.Info("Received CloseWorkflow signal, finishing workflows.", "BillID", bill.ID)
	bill.Status = domain.BillClosed
	return nil
}
