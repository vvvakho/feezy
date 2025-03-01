package api

import (
	"context"
	"time"

	"encore.dev/storage/sqldb"
	"github.com/vvvakho/feezy/domain"
)

var BillsDB = sqldb.NewDatabase("bills", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

// Define struct that implements BillStorage interface
type PostgresBillStorage struct {
	DB *sqldb.Database
}

func (s *PostgresBillStorage) AddToDB(ctx context.Context, bill domain.Bill) error {
	_, err := s.DB.Exec(ctx, `
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
	return err
}
