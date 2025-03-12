package workflows

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vvvakho/feezy/billing/service/domain"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

// BillWorkflow is a Temporal workflow that represents a stateful, long-running
// bill instance, beginning at bill creation and armed with signal and update receptors for
// processing bill events, such as adding or removing items, or querying and closing bill.
func BillWorkflow(ctx workflow.Context, bill *domain.Bill) (*domain.Bill, error) {
	// Initialize workflow with context, selector, and logger
	ctx, selector, logger, err := initWorkflow(ctx, bill)
	if err != nil {
		return bill, err
	}

	// Asynchronously add bill to `open_bills` DB
	// 1. Useful for fast querying of whether bill is open
	// 2. Minimizes cost implications of Temporal actions
	requestID := uuid.NewString()
	err = workflow.ExecuteActivity(ctx, AddOpenBillToDB, bill, requestID).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("Error saving bill to open_bills database: %v", err)
	}

	// Start listening for bill events
	for {
		// If bill status is closed, immediately finish work
		if bill.Status == domain.BillClosed {
			logger.Info("Bill closed, finishing workflows.", "BillID", bill.ID)
			break
		}

		// Process bill event signals and updates
		selector.Select(ctx)
	}

	return bill, nil
}

// Initialize the workflow with context, activity options, mutex, selector, logger.
func initWorkflow(ctx workflow.Context, bill *domain.Bill) (workflow.Context, workflow.Selector, log.Logger, error) {
	logger := workflow.GetLogger(ctx)

	bill.CreatedAt = time.Now()
	bill.UpdatedAt = time.Now()

	// Create a mutex for safe concurrency during requests
	mu := workflow.NewMutex(ctx)

	// Register handler for GetBill
	if err := workflow.SetQueryHandler(ctx, "getBill", func(input []byte) (*domain.Bill, error) {
		return bill, nil
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("SetQueryHandler failed: %v", err)
	}

	// Add custom activitiy options to context
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Register the Update handler for closing the bill
	err := HandleCloseBillUpdate(ctx, mu, bill, logger)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error registering handler for CloseBillUpdate: %v", err)
	}

	// Set up channels for receiving signals
	addLineItemChan := workflow.GetSignalChannel(ctx, AddLineItemRoute.Name)
	removeLineItemChan := workflow.GetSignalChannel(ctx, RemoveLineItemRoute.Name)
	closeBillChan := workflow.GetSignalChannel(ctx, CloseBillRoute.Name)
	closeWorkflowChan := workflow.GetSignalChannel(ctx, CloseWorkflowRoute.Name)
	selector := workflow.NewSelector(ctx)

	// Register handlers for processing signals
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
