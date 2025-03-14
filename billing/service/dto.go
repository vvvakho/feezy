package billing

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vvvakho/feezy/billing/service/domain"
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
	if err != nil || req.UserID == "" {
		return fmt.Errorf("Invalid UserID: %v", err)
	}
	_, err = domain.IsValidCurrency(req.Currency)
	if err != nil {
		return fmt.Errorf("Invalid Currency: %v", err)
	}
	return nil
}

type GetBillRequest struct {
	ID string `json:"id"`
}

type GetBillResponse struct {
	ID        string        `json:"id"`
	Items     []domain.Item `json:"items"`
	Total     domain.Money  `json:"total"`
	Status    domain.Status `json:"status"`
	UserID    string        `json:"user_id"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type AddLineItemRequest struct {
	ID           string       `json:"id"`
	Quantity     int64        `json:"quantity"`
	Description  string       `json:"description"`
	PricePerUnit domain.Money `json:"price_per_unit"`
}

type AddLineItemResponse struct {
	Message string `json:"message"`
}

func validateAddLineItemRequest(req *AddLineItemRequest) error {
	_, err := uuid.Parse(req.ID)
	if err != nil {
		return fmt.Errorf("Invalid ID: %v", err)
	}

	if req.PricePerUnit.Amount < 0 {
		return fmt.Errorf("Invalid price: %v", req.PricePerUnit)
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
	ID           string       `json:"id"`
	Quantity     int64        `json:"quantity"`
	Description  string       `json:"description"`
	PricePerUnit domain.Money `json:"price_per_unit"`
}

type RemoveLineItemResponse struct {
	Message string
}

func validateRemoveLineItemRequest(req *RemoveLineItemRequest) error {
	_, err := uuid.Parse(req.ID)
	if err != nil {
		return fmt.Errorf("Invalid item ID: %v", err)
	}

	if req.PricePerUnit.Amount < 0 {
		return fmt.Errorf("Invalid price: %v", req.PricePerUnit)
	}

	if req.Quantity < 1 {
		return fmt.Errorf("Invalid item quantity: %v", req.Quantity)
	}
	return nil
}

type CloseBillRequest struct {
	RequestID string `json:"request_id"`
}

type CloseBillResponse struct {
	Bill   *domain.Bill
	Status string
}

func validateCloseBillRequest(id string, req *CloseBillRequest) error {
	_, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("Invalid ID: %v", err)
	}

	if req.RequestID == "" {
		req.RequestID = uuid.NewString()
	}

	return nil
}
