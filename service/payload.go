package service

import (
	"time"

	"github.com/vvvakho/feezy/domain"
)

type CreateBillRequest struct {
	UserID   string `json:"user_id"`
	Currency string `json:"currency"`
}

type CreateBillResponse struct {
	ID string `json:"id"`
}

type GetBillRequest struct {
	ID string `json:"user_id"`
}

type GetBillResponse struct {
	ID        string        `json:"id"`
	Items     []domain.Item `json:"items"`
	Total     domain.Money  `json:"total"`
	Status    domain.Status `json:"status"`
	UserID    string        `json:"userId"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

type AddLineItemToBillRequest struct {
	ID           string
	Quantity     int64
	Description  string
	PricePerUnit domain.Money
}

type AddLineItemToBillResponse struct {
	Message string
}

type RemoveLineItemFromBillRequest struct {
	ID           string
	Quantity     int64
	Description  string
	PricePerUnit domain.Money
}

type RemoveLineItemFromBillResponse struct {
	Message string
}

type CloseBillRequest struct {
	ID        string `json:"user_id"`
	RequestID string `json:"request_id"`
}

type CloseBillResponse struct {
	Status string
}
