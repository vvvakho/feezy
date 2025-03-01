package workflow

import (
	"context"
	"fmt"

	"github.com/vvvakho/feezy/domain"
)

type Activities struct {
}

func (a *Activities) AddToDB(ctx context.Context, bill domain.Bill) error {
	// Simulating DB logic
	fmt.Printf("Saving bill %s to DB\n", bill.ID)

	// Call the database implementation
	// if err := db.AddToDB(ctx, bill); err != nil {
	// 	return err
	// }

	return nil
}
