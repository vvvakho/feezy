package billing

import (
	"context"
	"database/sql"
	"fmt"

	"encore.dev/storage/sqldb"
	"github.com/vvvakho/feezy/billing/service/domain"
)

var BillsDB = sqldb.NewDatabase("bills", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

type Repo struct {
	DB *sqldb.Database
}

func NewRepo() (*Repo, error) {
	return &Repo{DB: BillsDB}, nil
}

func (r *Repo) GetOpenBillFromDB(ctx context.Context, id string) (*domain.Bill, error) {
	// Start a new transaction
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback() // Ensure rollback is called if function exits early

	// Set the transaction isolation level to REPEATABLE READ
	_, err = tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ")
	if err != nil {
		return nil, fmt.Errorf("error setting transaction isolation level: %v", err)
	}

	query := `
		SELECT id, user_id, currency, status, created_at, updated_at
		FROM open_bills
		WHERE id = $1;
	`
	var bill domain.Bill
	row := tx.QueryRow(ctx, query, id)

	err = row.Scan(
		&bill.ID,
		&bill.UserID,
		&bill.Status,
		&bill.Total.Currency,
		&bill.CreatedAt,
		&bill.UpdatedAt,
	)

	// In case of errors the deferred rollback is activated
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("bill with ID %s not found in open_bills", id)
		}
		return nil, fmt.Errorf("error querying open_bills: %v", err)
	}

	// In the abscense of errors, commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error committing transaction: %v", err)
	}

	return &bill, nil
}

func (r *Repo) GetClosedBillFromDB(ctx context.Context, id string) (*domain.Bill, error) {
	// Start a new transaction
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback() // Ensure rollback is called if function exits early

	// Set the transaction isolation level to REPEATABLE READ
	_, err = tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ")
	if err != nil {
		return nil, fmt.Errorf("error setting transaction isolation level: %v", err)
	}

	query := `
		SELECT id, user_id, status, total_amount, currency, created_at, updated_at, closed_at
		FROM closed_bills
		WHERE id = $1;
	`

	var bill domain.Bill
	row := tx.QueryRow(ctx, query, id)

	err = row.Scan(
		&bill.ID,
		&bill.UserID,
		&bill.Status,
		&bill.Total.Amount,
		&bill.Total.Currency,
		&bill.CreatedAt,
		&bill.UpdatedAt,
		&bill.ClosedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("bill with ID %s not found in closed_bills", id)
		}
		return nil, fmt.Errorf("error querying closed_bills: %v", err)
	}

	// In the absence of errors, commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error committing transaction: %v", err)
	}

	return &bill, nil
}

func (r *Repo) GetClosedBillItemsFromDB(ctx context.Context, billID string) ([]domain.Item, error) {
	// Validate the billID
	if billID == "" {
		return nil, fmt.Errorf("billID cannot be empty")
	}

	// Start a new transaction
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback() // Ensure rollback is called if function exits early

	// Set the transaction isolation level to REPEATABLE READ
	_, err = tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ")
	if err != nil {
		return nil, fmt.Errorf("error setting transaction isolation level: %v", err)
	}

	if billID == "" {
		return nil, fmt.Errorf("billID cannot be empty")
	}

	// Query the database for items associated with the given billID
	rows, err := tx.Query(ctx, `
		SELECT item_id, description, quantity, unit_price, currency
		FROM closed_bills_items
		WHERE bill_id = $1
	`, billID)
	if err != nil {
		return nil, fmt.Errorf("error querying closed_bills_items: %v", err)
	}
	defer rows.Close()

	// Iterate through the rows and populate the items slice
	var items []domain.Item
	for rows.Next() {
		var item domain.Item
		var pricePerUnit domain.Money

		err := rows.Scan(
			&item.ID,
			&item.Description,
			&item.Quantity,
			&pricePerUnit.Amount,
			&pricePerUnit.Currency,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		// Assign the pricePerUnit to the item
		item.PricePerUnit = pricePerUnit

		// Append the item to the items slice
		items = append(items, item)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	// In the absence of errors, commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error committing transaction: %v", err)
	}

	return items, nil
}
