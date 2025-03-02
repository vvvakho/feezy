package workflow

import (
	"github.com/vvvakho/feezy/domain"
	"go.temporal.io/sdk/workflow"
)

func BillWorkflow(ctx workflow.Context, bill domain.Bill) error {
	// Initiate workflow with context, logger, error channel
	ctx, logger, errCh, err := initWorkflow(ctx, &bill)
	if err != nil {
		return err //TODO: fatal
	}

	// Asynchronously add bill to DB
	// addBillToDB(ctx, &bill, logger)

	// Set up handlers for signals
	addLineItemChan := workflow.GetSignalChannel(ctx, AddLineItemRoute.Name)
	removeLineItemChan := workflow.GetSignalChannel(ctx, RemoveLineItemRoute.Name)
	closeBillChan := workflow.GetSignalChannel(ctx, CloseBillRoute.Name)
	selector := workflow.NewSelector(ctx)

	registerSignalHandlers(
		ctx,
		selector,
		addLineItemChan,
		removeLineItemChan,
		closeBillChan,
		&bill,
		logger,
	)

	registerErrorHandler(ctx, selector, errCh, logger)

	// Start listening for events
	for {
		selector.Select(ctx)

		// Finish workflow when bill is closed
		if bill.Status == domain.BillClosed {
			logger.Info("Bill closed, finishing workflow.", "BillID", bill.ID)
			break
		}
	}

	return nil
}
