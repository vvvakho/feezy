package billing

import (
	"slices"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func BillWorkflow(ctx workflow.Context, bill Bill) error {
	// initiate logger
	logger := workflow.GetLogger(ctx)

	err := workflow.SetQueryHandler(ctx, "getBill", func(input []byte) (*Bill, error) {
		return &bill, nil
	})
	if err != nil {
		logger.Info("SetQueryHandler failed.", "Error", err)
		return err
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

	// start db addition asynchronously
	workflow.Go(ctx, func(ctx workflow.Context) {
		err := workflow.ExecuteActivity(ctx, "AddToDB", bill).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to add bill to DB", "Error", err)
		}
	})

	// set up handlers for signals
	addLineItemChan := workflow.GetSignalChannel(ctx, "addLineItem")
	removeLineItemChan := workflow.GetSignalChannel(ctx, "removeLineItem")
	closeBillChan := workflow.GetSignalChannel(ctx, "closeBill")
	billClosed := false

	selector := workflow.NewSelector(ctx)

	// register a handler to process the addLineItem signal
	selector.AddReceive(addLineItemChan, func(c workflow.ReceiveChannel, _ bool) {
		var addSignal AddItemSignal
		c.Receive(ctx, &addSignal)

		if err != nil {
			logger.Error("Invalid signal type %v", err)
			return
		}

		lineItem := addSignal.LineItem
		bill.AddLineItem(lineItem)

		// currently looking for an alternative to avoid bottlenecking worker
		// workflow.Go(ctx, func(ctx workflow.Context) {
		// 	err := workflow.ExecuteActivity(ctx, "UpdateBillTotalInDB", bill.ID, bill.Total).Get(ctx, nil)
		// 	if err != nil {
		// 		logger.Error("Failed to update bill total in DB", "Error", err)
		// 	}
		// })
	})

	// register a handler to process the removeLineItem signal
	selector.AddReceive(removeLineItemChan, func(c workflow.ReceiveChannel, _ bool) {
		var removeSignal RemoveItemSignal
		c.Receive(ctx, &removeSignal)

		if err != nil {
			logger.Error("Invalid signal type: %v")
			return
		}

		lineItem := removeSignal.LineItem
		bill.RemoveLineItem(lineItem)

		// currently looking for an alternative to avoid bottlenecking worker
		// err = workflow.ExecuteActivity(ctx, UpdateBillTotalInDB, state.ID, state.Total).Get(ctx, nil)
		// if err != nil {
		//   logger.Error("Failed to update bill total in DB", "Error", err)
		// }
	})

	selector.AddReceive(closeBillChan, func(c workflow.ReceiveChannel, _ bool) {
		var signal any
		c.Receive(ctx, &signal)

		billClosed = true

		return
		//TODO: actions to finalize, perhaps in workflow.Go routine ?
	})

	for {
		selector.Select(ctx)
		if billClosed {
			logger.Info("Bill closed, finishing workflow.", "BillID", bill.ID)
			break
		}
	}

	return nil
}

func (b *Bill) AddLineItem(item Item) {
	for i := range b.Items {
		if b.Items[i].ID != item.ID {
			continue
		}

		b.Items[i].Quantity += item.Quantity
		return
	}

	b.Items = append(b.Items, item)
}

func (b *Bill) RemoveLineItem(item Item) {
	for i := range b.Items {
		if b.Items[i].ID != item.ID {
			continue
		}

		b.Items[i].Quantity -= item.Quantity
		if b.Items[i].Quantity <= 0 {
			b.Items = slices.Delete(b.Items, i, i+1)
		}
		break
	}
}
