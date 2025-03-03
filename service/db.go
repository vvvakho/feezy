package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/vvvakho/feezy/domain"
)

func (s *Service) GetOpenBillFromDB(ctx context.Context, id string) (*domain.Bill, error) {
	query := `
		SELECT id, user_id, currency, status, created_at, updated_at
		FROM open_bills
		WHERE id = $1;
	`
	var bill domain.Bill
	row := s.DBencore.QueryRow(ctx, query, id)

	err := row.Scan(
		&bill.ID,
		&bill.UserID,
		&bill.Status,
		&bill.Total.Currency,
		&bill.CreatedAt,
		&bill.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("bill with ID %s not found in open_bills", id)
		}
		return nil, fmt.Errorf("error querying open_bills: %v", err)
	}

	return &bill, nil
}

func (s *Service) GetClosedBillFromDB(ctx context.Context, id string) (*domain.Bill, error) {
	query := `
		SELECT id, user_id, status, total_amount, currency, created_at, updated_at, closed_at
		FROM closed_bills
		WHERE id = $1;
	`

	var bill domain.Bill
	row := s.DBencore.QueryRow(ctx, query, id)

	err := row.Scan(
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

	return &bill, nil
}
