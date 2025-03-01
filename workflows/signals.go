package billing

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func initWorkflow(ctx workflow.Context, bill *Bill) (workflow.Context, log.Logger, workflow.Channel, error) {
	logger := workflow.GetLogger(ctx)
	errCh := workflow.NewChannel(ctx)

	// Register handler for GetBill
	if err := workflow.SetQueryHandler(ctx, "getBill", func(input []byte) (*Bill, error) {
		return bill, nil
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("SetQueryHandler failed: %v", err) //TODO: double check when to fatal vs log
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute, //TODO: investigate closer
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			MaximumInterval:    time.Minute,
			BackoffCoefficient: 2,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	return ctx, logger, errCh, nil
}

func registerSignalHandlers(
	ctx workflow.Context,
	selector workflow.Selector,
	addLineItemChan, removeLineItemChan, closeBillChan workflow.ReceiveChannel,
	bill *Bill,
	logger log.Logger,
) {

	// Register a handler to add line item
	selector.AddReceive(addLineItemChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := handleAddLineItemSignal(ctx, c, bill, logger); err != nil {
			logger.Error("Error adding item to bill", "Error", err)
			// errCh.Send(ctx, fmt.Errorf("Error adding item to bill: %v", err))
		}
	})

	// Register a handler to remove line item
	selector.AddReceive(removeLineItemChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := handleRemoveLineItemSignal(ctx, c, bill, logger); err != nil {
			logger.Error("Error removing item from bill", "Error", err)
			// errCh.Send(ctx, fmt.Errorf("Error removing item from bill: %v", err))
		}
	})

	// Register a handler to close bill
	selector.AddReceive(closeBillChan, func(c workflow.ReceiveChannel, _ bool) {
		if err := handleCloseBillSignal(ctx, c, bill, logger); err != nil {
			logger.Error("Error closing bill", "Error", err)
			// errCh.Send(ctx, fmt.Errorf("Error removing item from bill: %v", err))
		}
	})

}

func registerErrorHandler(
	ctx workflow.Context,
	selector workflow.Selector,
	errorChannel workflow.Channel,
	logger log.Logger,
) {
	// Register a handler to catch asynch errors
	selector.AddReceive(errorChannel, func(c workflow.ReceiveChannel, _ bool) {
		var err error
		c.Receive(ctx, &err)
		if err != nil {
			logger.Error("Asynchronous operation error", "Error", err)
		}
	})
}

func handleAddLineItemSignal(ctx workflow.Context, c workflow.ReceiveChannel, bill *Bill, logger log.Logger) error {
	var addSignal AddItemSignal
	c.Receive(ctx, &addSignal)

	lineItem := addSignal.LineItem
	if err := bill.addLineItem(lineItem); err != nil {
		return err
	}

	if err := bill.calculateTotal(); err != nil {
		return err
	}

	// Update bill total in DB asynchronously
	addBillToDB(ctx, bill, logger)

	return nil
}

func handleRemoveLineItemSignal(ctx workflow.Context, c workflow.ReceiveChannel, bill *Bill, logger log.Logger) error {
	var removeSignal RemoveItemSignal
	c.Receive(ctx, &removeSignal)

	lineItem := removeSignal.LineItem
	if err := bill.removeLineItem(lineItem); err != nil {
		return err
	}

	if err := bill.calculateTotal(); err != nil {
		return err
	}

	// Update bill total in DB asynchronously
	addBillToDB(ctx, bill, logger)
	return nil
}

func handleCloseBillSignal(ctx workflow.Context, c workflow.ReceiveChannel, bill *Bill, logger log.Logger) error {
	var closeSignal CloseBillSignal
	c.Receive(ctx, &closeSignal)

	err := mapstructure.Decode(closeSignal, &closeSignal)
	if err != nil {
		return fmt.Errorf("Invalid signal type: %v", err)
	}

	if err := bill.calculateTotal(); err != nil {
		return fmt.Errorf("Error calculating bill total: %v", err)
	}

	bill.Status = BillClosed

	// Update bill total in DB asynchronously
	addBillToDB(ctx, bill, logger)

	return nil
}
