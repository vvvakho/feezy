package workflow

import (
	"context"
	"fmt"

	db "github.com/vvvakho/feezy/db/postgres"
	"github.com/vvvakho/feezy/domain"
)

// Define an interface for database operations
type BillStorage interface {
	AddToDB(ctx context.Context, bill domain.Bill) error
}

type Activities struct {
	DB *db.PostgresBillStorage
}

func (a *Activities) AddToDB(ctx context.Context, bill domain.Bill) error {
	// Simulating DB logic
	fmt.Printf("Saving bill %s to DB\n", bill.ID)

	// Call the database implementation
	if err := a.DB.AddToDB(ctx, bill); err != nil {
		return err
	}

	return nil
}
