package billing

import (
	"errors"
	"slices"
	"time"

	"github.com/mitchellh/mapstructure"
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

		lineItem := addSignal.LineItem
		if err := bill.AddLineItem(lineItem); err != nil {
			logger.Error("Error adding item to bill", "Error", err)
			return
		}

		if err := bill.CalculateTotal(); err != nil {
			logger.Error("Error calculating bill total", "Error", err)
			return
		}

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

		lineItem := removeSignal.LineItem
		if err := bill.RemoveLineItem(lineItem); err != nil {
			logger.Error("Error removing item from bill", "Error", err)
			return
		}

		if err := bill.CalculateTotal(); err != nil {
			logger.Error("Error calculating bill total", "Error", err)
			return
		}

		// currently looking for an alternative to avoid bottlenecking worker
		// err = workflow.ExecuteActivity(ctx, UpdateBillTotalInDB, state.ID, state.Total).Get(ctx, nil)
		// if err != nil {
		//   logger.Error("Failed to update bill total in DB", "Error", err)
		// }
	})

	selector.AddReceive(closeBillChan, func(c workflow.ReceiveChannel, _ bool) {
		var closeSignal CloseBillSignal
		c.Receive(ctx, &closeSignal)

		err := mapstructure.Decode(closeSignal, &closeSignal)
		if err != nil {
			logger.Error("Invalid signal type", "Error", err)
			return
		}

		if err := bill.CalculateTotal(); err != nil {
			logger.Error("Error calculating bill total", "Error", err)
			return
		}

		bill.Status = Closed
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

func (b *Bill) AddLineItem(itemToAdd Item) error {
	for i, itemInBill := range b.Items {
		if itemInBill.ID == itemToAdd.ID {
			if itemInBill.PricePerUnit != itemToAdd.PricePerUnit {
				return errors.New("Price of item has changed, please use new UUID")
			}
			b.Items[i].Quantity += itemToAdd.Quantity
			return nil
		}
	}
	b.Items = append(b.Items, itemToAdd)

	return nil
}

func (b *Bill) RemoveLineItem(itemToRemove Item) error {
	for i, itemInBill := range b.Items {
		if itemInBill.ID == itemToRemove.ID {
			b.Items[i].Quantity -= itemToRemove.Quantity
			if b.Items[i].Quantity <= 0 {
				b.Items = slices.Delete(b.Items, i, i+1)
			}
		}
	}

	return nil
}

func (b *Bill) CalculateTotal() error {
	var total minorUnit
	for _, v := range b.Items {
		amount := v.PricePerUnit.Amount         // 275 gel
		fromCurrency := v.PricePerUnit.Currency // gel
		toCurrency := b.Total.Currency          // usd

		unitPrice, err := convert(toCurrency, fromCurrency, amount) // 100
		if err != nil {
			return err
		}

		total += unitPrice * minorUnit(v.Quantity) // 100 * 1
	}
	b.Total.Amount = total

	return nil
}
