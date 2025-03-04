package workflows

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vvvakho/feezy/billing/service/domain"
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

type Repo struct {
	DB *sql.DB
}

func (r *Repo) AddOpenBillToDB(ctx context.Context, bill *domain.Bill, requestID *string) error {
	tx, err := r.DB.Begin()
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

func (r *Repo) AddClosedBillToDB(ctx context.Context, bill *domain.Bill, requestID *string) error {
	// Validate requestID before initiating transaction
	if requestID == nil {
		return temporal.NewNonRetryableApplicationError("requestID cannot be nil", "InvalidRequestError", nil)
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Error starting transaction: %v", err)
	}
	defer tx.Rollback()

	// Attempt to move the bill from Temporal Workflow into the closed_bills table in database
	res, err := tx.ExecContext(ctx, `
		INSERT INTO closed_bills (id, user_id, status, total_amount, currency, created_at, updated_at, closed_at, request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) 
		DO UPDATE SET 
			status = EXCLUDED.status,
			total_amount = EXCLUDED.total_amount,
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
		*requestID,
	)

	if err != nil {
		if isUserInputError(err) {
			// Mark this error as non-retryable for Temporal
			return temporal.NewNonRetryableApplicationError("Invalid input error", "UserInputError", err)
		}
		return fmt.Errorf("Error inserting/updating db: %v", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %v", err)
	}

	// If no rows affected, check if the existing request_id matches
	if rowsAffected == 0 {
		var existingRequestID string
		err := tx.QueryRowContext(ctx, "SELECT request_id FROM closed_bills WHERE id = $1", bill.ID).Scan(&existingRequestID)
		if err != nil {
			return fmt.Errorf("error checking existing request_id: %v", err)
		}

		if existingRequestID == *requestID {
			// Idempotent request: already processed successfully
			return tx.Commit() // Commit to signal successful idempotent operation
		} else {
			// Bill closed with a different requestID: non-retryable error
			return temporal.NewNonRetryableApplicationError(
				"bill already closed with a different request",
				"BillAlreadyClosedError",
				nil,
			)
		}
	}

	// Attempt to move the bill items from Temporal Workflow into the closed_bills_items table in database
	for _, item := range bill.Items {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO closed_bills_items (id, bill_id, item_id, description, quantity, unit_price, currency)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) 
			DO UPDATE SET 
				description = EXCLUDED.description,
				quantity = EXCLUDED.quantity,
				unit_price = EXCLUDED.unit_price,
				currency = EXCLUDED.currency;`,
			uuid.New(),
			bill.ID,
			item.ID,
			item.Description,
			item.Quantity,
			item.PricePerUnit.Amount,
			item.PricePerUnit.Currency,
		)
		if err != nil {
			return fmt.Errorf("Error inserting/updating closed_bills_items: %v", err)
		}
	}

	// Attempt to remove bill from Open Bills Database
	_, err = tx.ExecContext(ctx, `DELETE FROM open_bills WHERE id = $1`, bill.ID)
	if err != nil {
		return fmt.Errorf("Failed to remove from open_bills: %v", err)
	}

	// Commit the transaction
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
