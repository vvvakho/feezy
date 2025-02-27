package fees

import (
	"time"

	"github.com/mitchellh/mapstructure"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func BillWorkflow(ctx workflow.Context, state BillState) error {
	// initiate logger
	logger := workflow.GetLogger(ctx)

	err := workflow.SetQueryHandler(ctx, "getBill", func(input []byte) (*BillState, error) {
		return &state, nil
	})
	if err != nil {
		logger.Info("SetQueryHandler failed.", "Error", err)
		return err
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute, // does this terminate the operation entirely???
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			MaximumInterval:    time.Minute,
			BackoffCoefficient: 2,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// start db addition asynchronously
	var s *Service
	future := workflow.ExecuteActivity(ctx, s.AddBillToDB, state)

	// set up handlers for signals
	addLineItemChan := workflow.GetSignalChannel(ctx, "addLineItem")
	removeLineItemChan := workflow.GetSignalChannel(ctx, "removeLineItem")

	for {
		selector := workflow.NewSelector(ctx)

		selector.AddFuture(future, func(f workflow.Future) {
			err := f.Get(ctx, nil)
			if err != nil {
				logger.Error("Failed to add bill to DB", "Error", err)
			}
		})

		// register a handler for processing the addLineItem signal
		selector.AddReceive(addLineItemChan, func(c workflow.ReceiveChannel, _ bool) {
			var signal any
			c.Receive(ctx, &signal)

			var addSignal AddLineItemSignal
			err := mapstructure.Decode(signal, &addSignal)
			if err != nil {
				logger.Error("Invalid signal type %v", err)
				return
			}
			lineItem := addSignal.LineItem
			state.AddLineItem(lineItem)
		})

		// register a handler for processing the removeLineItem signal
		selector.AddReceive(removeLineItemChan, func(c workflow.ReceiveChannel, _ bool) {
			var signal any
			c.Receive(ctx, &signal)

			var removeSignal RemoveLineItemSignal
			err := mapstructure.Decode(signal, &removeSignal)
			if err != nil {
				logger.Error("Invalid signal type: %v")
				return
			}

			lineItem := removeSignal.LineItem
			state.RemoveLineItem(lineItem)
		})

	}
	return nil
}

func (b *BillState) AddLineItem(item LineItem) {
	for i := range b.Items {
		if b.Items[i].ID != b.ID {
			continue
		}

		b.Items[i].Quantity += item.Quantity
		return
	}

	b.Items = append(b.Items, item)
}

func (b *BillState) RemoveLineItem(item LineItem) {

}
