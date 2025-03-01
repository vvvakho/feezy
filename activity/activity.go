package activity

import (
	"context"
	"fmt"
	"time"

	"encore.dev/storage/sqldb"
	"github.com/vvvakho/feezy/domain"
)

type Activities struct {
	DB *sqldb.Database
}

func (a *Activities) AddToDB(ctx context.Context, bill domain.Bill, db *sqldb.Database) error {
	fmt.Printf("Saving bill %s to DB\n", bill.ID)
	_, err := a.DB.Exec(ctx, `
		INSERT INTO closed_bills (
			ID, UserID, Status, TotalAmount, Currency, CreatedAt, UpdatedAt, ClosedAt)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`,
		bill.ID,
		bill.UserID,
		bill.Status,
		bill.Total.Amount,
		bill.Total.Currency,
		bill.CreatedAt,
		bill.UpdatedAt,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("Error inserting into db: %v", err)
	}
	return nil
}
