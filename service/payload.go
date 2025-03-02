package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vvvakho/feezy/domain"
)

type CreateBillRequest struct {
	UserID   string `json:"user_id"`
	Currency string `json:"currency"`
}

type CreateBillResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

func validateCreateBillRequest(req *CreateBillRequest) error {
	_, err := uuid.Parse(req.UserID)
	if err != nil {
		return fmt.Errorf("Invalid UserID: %v", err)
	}
	_, err = domain.IsValidCurrency(req.Currency)
	if err != nil {
		return fmt.Errorf("Invalid Currency: %v", err)
	}
	return nil
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

type AddLineItemRequest struct {
	ID           string
	Quantity     int64
	Description  string
	PricePerUnit domain.Money
}

type AddLineItemResponse struct {
	Message string
}

func validateAddLineItemRequest(req *AddLineItemRequest) error {
	_, err := uuid.Parse(req.ID)
	if err != nil {
		return fmt.Errorf("Invalid ID: %v", err)
	}
	if err := validateAddLineItemRequest(req); err != nil {
		return fmt.Errorf("Invalid request parameters: %v", err)
	}
	if req.Quantity < 1 {
		return fmt.Errorf("Invalid item quantity: %v", req.Quantity)
	}
	_, err = domain.IsValidCurrency(req.PricePerUnit.Currency)
	if err != nil {
		return fmt.Errorf("Invalid currency %v", err)
	}

	return nil
}

type RemoveLineItemRequest struct {
	ID           string
	Quantity     int64
	Description  string
	PricePerUnit domain.Money
}

type RemoveLineItemResponse struct {
	Message string
}

func validateRemoveLineItemRequest(req *RemoveLineItemRequest) error {
	_, err := uuid.Parse(req.ID)
	if err != nil {
		return fmt.Errorf("Invalid item ID: %v", err)
	}

	return nil
}

type CloseBillRequest struct {
	ID        string `json:"user_id"`
	RequestID string `json:"request_id"`
}

type CloseBillResponse struct {
	Status string
}
