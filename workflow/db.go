package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/vvvakho/feezy/domain"
	"go.temporal.io/sdk/temporal"
)

type WorkerDB struct {
	DB *sql.DB
}

func (db *WorkerDB) AddClosedBillToDB(ctx context.Context, bill *domain.Bill, requestID *string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("Error starting transaction: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.Exec(`
		INSERT INTO closed_bills (id, user_id, status, total_amount, currency, created_at, updated_at, closed_at, request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) 
		DO UPDATE SET 
			status = CASE WHEN closed_bills.status <> EXCLUDED.status THEN EXCLUDED.status ELSE closed_bills.status END,
			total_amount = CASE WHEN closed_bills.total_amount <> EXCLUDED.total_amount THEN EXCLUDED.total_amount ELSE closed_bills.total_amount END,
			updated_at = now()
		WHERE closed_bills.request_id IS DISTINCT FROM EXCLUDED.request_id;
	`,
		bill.ID,
		bill.UserID,
		domain.BillClosed,
		bill.Total.Amount,
		bill.Total.Currency,
		bill.CreatedAt,
		time.Now(),
		time.Now(),
		requestID,
	)

	if err != nil {
		// Check if the error is a duplicate key error
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			// Mark this error as non-retryable for Temporal
			return temporal.NewNonRetryableApplicationError("duplicate request detected", "DuplicateRequestError", err)
		}
		return fmt.Errorf("Error inserting/updating db: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("Error committing transaction: %v", err)
	}

	return nil
}
