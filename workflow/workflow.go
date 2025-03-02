package workflow

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vvvakho/feezy/domain"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

func BillWorkflow(ctx workflow.Context, bill *domain.Bill) error {
	// Initiate workflow with context, logger, error channel
	ctx, logger, err := initWorkflow(ctx, bill)
	if err != nil {
		return err //TODO: fatal
	}

	// Asynchronously add bill to `open_bills` DB
	requestID := uuid.NewString()
	err = workflow.ExecuteActivity(ctx, AddOpenBillToDB, bill, requestID).Get(ctx, nil)
	if err != nil {
		return err
	}

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
		bill,
		logger,
	)

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
