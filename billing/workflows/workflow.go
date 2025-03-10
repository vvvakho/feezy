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
	// Initialize workflow with context, selector, and logger
	ctx, selector, logger, err := initWorkflow(ctx, bill)
	if err != nil {
		return bill, err
	}

	// Asynchronously add bill to `open_bills` DB
	requestID := uuid.NewString()
	err = workflow.ExecuteActivity(ctx, AddOpenBillToDB, bill, requestID).Get(ctx, nil)
	if err != nil {
		return bill, err
	}

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

func initWorkflow(ctx workflow.Context, bill *domain.Bill) (workflow.Context, workflow.Selector, log.Logger, error) {
	logger := workflow.GetLogger(ctx)

	bill.CreatedAt = time.Now()
	bill.UpdatedAt = time.Now()

	// Create a mutex for safe concurrency
	mu := workflow.NewMutex(ctx)

	// Register handler for GetBill
	if err := workflow.SetQueryHandler(ctx, "getBill", func(input []byte) (*domain.Bill, error) {
		return bill, nil
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("SetQueryHandler failed: %v", err)
	}

	// Add custom default activitiy options to context
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Register the Update handler for closing the bill
	err := HandleCloseBillUpdate(ctx, mu, bill, logger)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error registering handler for CloseBillUpdate: %v", err)
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
		mu,
		selector,
		addLineItemChan,
		removeLineItemChan,
		closeBillChan,
		closeWorkflowChan,
		bill,
		logger,
	)

	return ctx, selector, logger, nil
}
