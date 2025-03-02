package workflow

import (
	"context"

	"github.com/vvvakho/feezy/domain"
)

type Activities struct {
	Repository Repository
}

type Repository interface {
	AddClosedBillToDB(context.Context, *domain.Bill, *string) error
}

func (a *Activities) AddClosedBillToDB(ctx context.Context, bill *domain.Bill, requestID *string) error {
	return a.Repository.AddClosedBillToDB(ctx, bill, requestID)
}
