package billing

import (
	"context"
	"testing"
	"time"

	"encore.dev/et"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetOpenBillFromDB(t *testing.T) {
	ctx := context.Background()

	// Create a new test database
	testDB, err := et.NewTestDatabase(ctx, "bills")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	repo := Repo{DB: testDB}

	// Generate a valid UUID
	billID := uuid.New().String()
	userID := uuid.New().String()

	// Insert test data with a valid UUID
	_, err = testDB.Exec(ctx, `
        INSERT INTO open_bills (id, user_id, currency, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6);
    `, billID, userID, "USD", "open", time.Now(), time.Now())

	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	bill, err := repo.GetOpenBillFromDB(ctx, billID)

	assert.NoError(t, err)
	assert.NotNil(t, bill)
	assert.Equal(t, billID, bill.ID.String())
}
