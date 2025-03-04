package workflows

import (
	"context"
	"testing"
	"time"

	"encore.dev/et"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vvvakho/feezy/billing/service/domain"
)

func TestAddOpenBillToDB(t *testing.T) {
	ctx := context.Background()

	// Create a new test database
	testDB, err := et.NewTestDatabase(ctx, "bills")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	repo := Repo{DB: testDB.Stdlib()}
	activities := Activities{Repository: &repo}

	// Generate test data
	billID := uuid.New()
	userID := uuid.New()
	requestID := uuid.New().String()

	bill := &domain.Bill{
		ID:        billID,
		UserID:    userID,
		Total:     domain.Money{Currency: "USD"},
		CreatedAt: time.Now(),
	}

	// Execute the activity
	err = activities.AddOpenBillToDB(ctx, bill, &requestID)
	assert.NoError(t, err)

	// Verify that the bill was inserted
	var dbBill domain.Bill
	err = testDB.QueryRow(ctx, `SELECT id, user_id, currency, status FROM open_bills WHERE id = $1`, billID).Scan(
		&dbBill.ID, &dbBill.UserID, &dbBill.Total.Currency, &dbBill.Status,
	)

	assert.NoError(t, err)
	assert.Equal(t, billID, dbBill.ID)
	assert.Equal(t, userID, dbBill.UserID)
	assert.Equal(t, "USD", dbBill.Total.Currency)
	assert.Equal(t, domain.BillOpen, dbBill.Status) // Assuming "BillOpen" is the correct status.
}

func TestAddClosedBillToDB(t *testing.T) {
	ctx := context.Background()

	// Create a new test database
	testDB, err := et.NewTestDatabase(ctx, "bills")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	repo := Repo{DB: testDB.Stdlib()}
	activities := Activities{Repository: &repo}

	// Generate test data
	billID := uuid.New()
	userID := uuid.New()
	requestID := uuid.New().String()

	bill := &domain.Bill{
		ID:        billID,
		UserID:    userID,
		Total:     domain.Money{Amount: 100, Currency: "USD"},
		CreatedAt: time.Now(),
	}

	// Insert bill into open_bills before closing it
	_, err = testDB.Exec(ctx, `
		INSERT INTO open_bills (id, user_id, currency, status, created_at, updated_at, request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7);
	`, billID, userID, "USD", domain.BillOpen, time.Now(), time.Now(), requestID)

	assert.NoError(t, err)

	// Execute the activity to close the bill
	err = activities.AddClosedBillToDB(ctx, bill, &requestID)
	assert.NoError(t, err)

	// Verify that the bill was moved to `closed_bills`
	var dbBill domain.Bill
	err = testDB.QueryRow(ctx, `SELECT id, user_id, total_amount, currency, status FROM closed_bills WHERE id = $1`, billID).Scan(
		&dbBill.ID, &dbBill.UserID, &dbBill.Total.Amount, &dbBill.Total.Currency, &dbBill.Status,
	)

	assert.NoError(t, err)
	assert.Equal(t, billID, dbBill.ID)
	assert.Equal(t, userID, dbBill.UserID)
	assert.Equal(t, domain.MinorUnit(100), dbBill.Total.Amount)
	assert.Equal(t, "USD", dbBill.Total.Currency)
	assert.Equal(t, domain.BillClosed, dbBill.Status)

	// Verify that the bill was removed from `open_bills`
	var count int
	err = testDB.QueryRow(ctx, `SELECT COUNT(*) FROM open_bills WHERE id = $1`, billID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count) // Should be removed
}
