package activity

import (
	"context"
	"fmt"

	"github.com/vvvakho/feezy/api"
	"github.com/vvvakho/feezy/domain"
)

type Activities struct {
	Server *api.Server
}

func (a *Activities) AddToDB(ctx context.Context, bill domain.Bill) error {
	fmt.Printf("Saving bill %s to DB\n", bill.ID)
	a.Server.AddToDB(ctx, bill)
	// Call the database implementation

	return nil
}
