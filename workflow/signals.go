package workflow

import (
	"errors"
	"time"

	"github.com/vvvakho/feezy/domain"
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

func registerSignalHandlers(
	ctx workflow.Context,
	selector workflow.Selector,
	addLineItemChan, removeLineItemChan, closeBillChan workflow.ReceiveChannel,
	bill *domain.Bill,
	logger log.Logger,
) {

	// Register a handler to add line item
	selector.AddReceive(addLineItemChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := handleAddLineItemSignal(ctx, c, bill); err != nil {
			logger.Error("Adding item to bill", "Error", err)
		}
	})

	// Register a handler to remove line item
	selector.AddReceive(removeLineItemChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := handleRemoveLineItemSignal(ctx, c, bill); err != nil {
			logger.Error("Removing item from bill", "Error", err)
		}
	})

	// Register a handler to close bill
	selector.AddReceive(closeBillChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := handleCloseBillSignal(ctx, c, bill, logger); err != nil {
			logger.Error("Closing bill", "Error", err)
		}
	})

}

func handleAddLineItemSignal(ctx workflow.Context, c workflow.ReceiveChannel, bill *domain.Bill) error {
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

	// Update bill total in DB asynchronously
	// addBillToDB(ctx, bill, logger)

	return nil
}

func handleRemoveLineItemSignal(ctx workflow.Context, c workflow.ReceiveChannel, bill *domain.Bill) error {
	var removeSignal RemoveItemSignal
	c.Receive(ctx, &removeSignal)

	lineItem := removeSignal.LineItem
	if err := bill.RemoveLineItem(lineItem); err != nil {
		return err
	}

	if err := bill.CalculateTotal(); err != nil {
		return err
	}

	bill.UpdatedAt = time.Now()

	// Update bill total in DB asynchronously
	// addBillToDB(ctx, bill, logger)
	return nil
}

func handleCloseBillSignal(ctx workflow.Context, c workflow.ReceiveChannel, bill *domain.Bill, logger log.Logger) error {
	for {
		var closeSignal CloseBillSignal
		c.Receive(ctx, &closeSignal)

		if err := bill.CalculateTotal(); err != nil {
			logger.Error("Error calculating bill total", "Error", err)
			continue // Keep waiting for a valid signal
		}

		bill.Status = domain.BillClosing

		// Execute activity with retry logic
		err := workflow.ExecuteActivity(ctx, AddClosedBillToDB, bill, closeSignal.RequestID).Get(ctx, nil)
		if err != nil {
			var appErr *temporal.ApplicationError
			if errors.As(err, &appErr) && appErr.Type() == "DuplicateRequestError" {
				logger.Warn("Duplicate request detected, waiting for new request")
				continue // Keep waiting for a correct request ID
			}
			logger.Error("Error executing AddClosedBillToDB activity", "Error", err)
			return err // For other errors, fail the workflow
		}

		// Successfully closed the bill, exit loop
		bill.Status = domain.BillClosed
		bill.UpdatedAt = time.Now()

		logger.Info("Bill successfully saved as closed in DB", "BillID", bill.ID)
		break
	}

	return nil
}
