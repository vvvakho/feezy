package workflow

import (
	"context"
	"time"

	"github.com/vvvakho/feezy/domain"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var AddOpenBillToDB string = "AddOpenBillToDB"
var AddClosedBillToDB string = "AddClosedBillToDB"

var ao = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Second,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		MaximumInterval:    time.Minute,
		BackoffCoefficient: 2,
	},
}

type Activities struct {
	Repository Repository
}

type Repository interface {
	AddOpenBillToDB(context.Context, *domain.Bill, *string) error
	AddClosedBillToDB(context.Context, *domain.Bill, *string) error
}

func (a *Activities) AddOpenBillToDB(ctx context.Context, bill *domain.Bill, requestID *string) error {
	return a.Repository.AddOpenBillToDB(ctx, bill, requestID)
}

func (a *Activities) AddClosedBillToDB(ctx context.Context, bill *domain.Bill, requestID *string) error {
	return a.Repository.AddClosedBillToDB(ctx, bill, requestID)
}
