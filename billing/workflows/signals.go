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

func registerSignalHandlers(
	ctx workflow.Context,
	selector workflow.Selector,
	addLineItemChan,
	removeLineItemChan,
	closeBillChan,
	closeWorkflowChan workflow.ReceiveChannel,
	bill *domain.Bill,
	logger log.Logger,
) {

	// Register a handler to add line item
	selector.AddReceive(addLineItemChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := HandleAddLineItemSignal(ctx, c, bill); err != nil {
			logger.Error("Adding item to bill", "Error", err)
		}
	})

	// Register a handler to remove line item
	selector.AddReceive(removeLineItemChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := HandleRemoveLineItemSignal(ctx, c, bill); err != nil {
			logger.Error("Removing item from bill", "Error", err)
		}
	})

	// Register a handler to close bill
	selector.AddReceive(closeBillChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := HandleCloseBillSignal(ctx, c, bill, logger); err != nil {
			logger.Error("Closing bill", "Error", err)
		}
	})

	// Register a handler to close workflow
	selector.AddReceive(closeWorkflowChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := HandleCloseWorkflowSignal(ctx, c, bill, logger); err != nil {
			logger.Error("Closing workflow", "Error", err)
		}
	})
}

func HandleAddLineItemSignal(ctx workflow.Context, c workflow.ReceiveChannel, bill *domain.Bill) error {
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

func HandleRemoveLineItemSignal(ctx workflow.Context, c workflow.ReceiveChannel, bill *domain.Bill) error {
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

func HandleCloseBillSignal(ctx workflow.Context, c workflow.ReceiveChannel, bill *domain.Bill, logger log.Logger) error {
	for {
		// If the bill is already closed, ignore further signals
		if bill.Status == domain.BillClosed {
			logger.Warn("Received close bill signal, but bill is already closed", "BillID", bill.ID)
			return fmt.Errorf("Bill already closed")
		}

		bill.Status = domain.BillClosing
		bill.UpdatedAt = time.Now()

		var closeSignal CloseBillSignal
		c.Receive(ctx, &closeSignal)

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

		// Execute AddClosedBillToDB activity
		err := workflow.ExecuteActivity(ctx, AddClosedBillToDB, bill, closeSignal.RequestID).Get(ctx, nil)
		if err != nil {
			var appErr *temporal.ApplicationError
			if errors.As(err, &appErr) {
				if appErr.Type() == "DuplicateRequestError" {
					logger.Warn("Duplicate close request detected, ignoring", "RequestID", closeSignal.RequestID)
					return nil
				} else if appErr.Type() == "UserInputError" {
					logger.Error("Invalid input, rejecting close request", "Error", appErr)
					bill.Status = domain.BillOpen // Reset to open state
					return err
				}
			}
			logger.Error("Error executing AddClosedBillToDB activity", "Error", err)
			return err // Fail workflows for other errors
		}

		// Successfully closed the bill, exit loop
		bill.Status = domain.BillClosed

		logger.Info("Bill successfully saved as closed in DB", "BillID", bill.ID)
		break
	}

	return nil
}

func HandleCloseBillUpdate(ctx workflow.Context, bill *domain.Bill, logger log.Logger) error {
	err := workflow.SetUpdateHandler(ctx, "CloseBillUpdate", func(ctx workflow.Context, requestID string) (*domain.Bill, error) {
		if bill.Status == domain.BillClosed {
			logger.Warn("Received close bill update, but bill is already closed", "BillID", bill.ID)
			return &domain.Bill{}, fmt.Errorf("Bill already closed")
		}

		bill.Status = domain.BillClosing
		bill.UpdatedAt = time.Now()

		if err := bill.CalculateTotal(); err != nil {
			logger.Error("Error calculating bill total", "Error", err)
			bill.Status = domain.BillOpen
			return &domain.Bill{}, fmt.Errorf("Error closing bill: %v", err)
		}

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

		err := workflow.ExecuteActivity(ctx, AddClosedBillToDB, bill, requestID).Get(ctx, nil)
		if err != nil {
			var appErr *temporal.ApplicationError
			if errors.As(err, &appErr) {
				if appErr.Type() == "DuplicateRequestError" {
					logger.Warn("Duplicate close request detected, ignoring", "RequestID", requestID)
					return &domain.Bill{}, nil
				} else if appErr.Type() == "UserInputError" {
					logger.Error("Invalid input, rejecting close request", "Error", appErr)
					bill.Status = domain.BillOpen
					return &domain.Bill{}, err
				}
			}
			logger.Error("Error executing AddClosedBillToDB activity", "Error", err)
			return &domain.Bill{}, err
		}

		bill.Status = domain.BillClosed
		logger.Info("Bill successfully saved as closed in DB", "BillID", bill.ID)
		return bill, nil
	})
	return err
}

func HandleCloseWorkflowSignal(ctx workflow.Context, c workflow.ReceiveChannel, bill *domain.Bill, logger log.Logger) error {
	var closeWFSignal CloseWorkflowSignal
	c.Receive(ctx, &closeWFSignal)

	logger.Info("Received CloseWorkflow signal, finishing workflows.", "BillID", bill.ID)
	bill.Status = domain.BillClosed
	return nil
}
