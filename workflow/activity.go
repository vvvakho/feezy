package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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

type DB struct {
	DBworker *sql.DB
}

func (db *DB) AddOpenBillToDB(ctx context.Context, bill *domain.Bill, requestID *string) error {
	tx, err := db.DBworker.Begin()
	if err != nil {
		return fmt.Errorf("Error starting transaction: %v", err)
	}

	_, err = tx.Exec(`
		INSERT INTO open_bills (id, user_id, status, currency, created_at, updated_at, request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id)
		DO UPDATE SET 
			status = CASE WHEN open_bills.status <> EXCLUDED.status THEN EXCLUDED.status ELSE open_bills.status END,
			updated_at = now()
		WHERE open_bills.request_id IS DISTINCT FROM EXCLUDED.request_id;
	`,
		bill.ID,
		bill.UserID,
		domain.BillOpen,
		bill.Total.Currency,
		bill.CreatedAt,
		time.Now(),
		requestID,
	)

	if err != nil {
		tx.Rollback()
		if isUserInputError(err) {
			// Mark as a non-retryable error in Temporal
			return temporal.NewNonRetryableApplicationError("Invalid input error", "UserInputError", err)
		}
		return fmt.Errorf("Error inserting/updating db: %v", err)
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return fmt.Errorf("Error committing transaction: %v", err)
	}

	return nil
}

func (db *DB) AddClosedBillToDB(ctx context.Context, bill *domain.Bill, requestID *string) error {
	tx, err := db.DBworker.Begin()
	if err != nil {
		return fmt.Errorf("Error starting transaction: %v", err)
	}

	// Check if the bill is already closed with this requestID
	var existingRequestID *string
	err = tx.QueryRow("SELECT request_id FROM closed_bills WHERE id = $1", bill.ID).Scan(&existingRequestID)
	if err == nil && existingRequestID != nil && *existingRequestID == *requestID {
		tx.Rollback()
		return temporal.NewNonRetryableApplicationError("Duplicate request", "DuplicateRequestError", nil)
	}

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
		tx.Rollback()
		if isUserInputError(err) {
			// Mark this error as non-retryable for Temporal
			return temporal.NewNonRetryableApplicationError("Invalid input error", "UserInputError", err)
		}
		return fmt.Errorf("Error inserting/updating db: %v", err)
	}

	// Remove from Open Bills Database
	_, err = tx.Exec(`DELETE FROM open_bills WHERE id = $1`, bill.ID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("Failed to remove from open_bills: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("Error committing transaction: %v", err)
	}

	return nil
}

func isUserInputError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "duplicate key value")
	// We can define other User related errors below
}
