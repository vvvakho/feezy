package workflows

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vvvakho/feezy/billing/service/domain"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

func BillWorkflow(ctx workflow.Context, bill *domain.Bill) (*domain.Bill, error) {
	// Initiate workflows with context, logger, error channel
	ctx, logger, err := initWorkflow(ctx, bill)
	if err != nil {
		return bill, err //TODO: fatal
	}

	// Asynchronously add bill to `open_bills` DB
	requestID := uuid.NewString()
	err = workflow.ExecuteActivity(ctx, AddOpenBillToDB, bill, requestID).Get(ctx, nil)
	if err != nil {
		return bill, err
	}

	// Register the Update handler for closing the bill
	err = HandleCloseBillUpdate(ctx, bill, logger)
	if err != nil {
		return bill, err
	}

	// Set up channels for signals
	addLineItemChan := workflow.GetSignalChannel(ctx, AddLineItemRoute.Name)
	removeLineItemChan := workflow.GetSignalChannel(ctx, RemoveLineItemRoute.Name)
	closeBillChan := workflow.GetSignalChannel(ctx, CloseBillRoute.Name)
	closeWorkflowChan := workflow.GetSignalChannel(ctx, CloseWorkflowRoute.Name)
	selector := workflow.NewSelector(ctx)

	// Register handlers for signals
	registerSignalHandlers(
		ctx,
		selector,
		addLineItemChan,
		removeLineItemChan,
		closeBillChan,
		closeWorkflowChan,
		bill,
		logger,
	)

	// Start listening for events
	for {

		// Finish workflows when bill is closed
		if bill.Status == domain.BillClosed {
			logger.Info("Bill closed, finishing workflows.", "BillID", bill.ID)
			break
		}

		selector.Select(ctx)
	}

	return bill, nil
}

func initWorkflow(ctx workflow.Context, bill *domain.Bill) (workflow.Context, log.Logger, error) {
	logger := workflow.GetLogger(ctx)

	bill.CreatedAt = time.Now()
	bill.UpdatedAt = time.Now()

	// Register handler for GetBill
	if err := workflow.SetQueryHandler(ctx, "getBill", func(input []byte) (*domain.Bill, error) {
		return bill, nil
	}); err != nil {
		return nil, nil, fmt.Errorf("SetQueryHandler failed: %v", err) //TODO: double check when to fatal vs log
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	return ctx, logger, nil
}
