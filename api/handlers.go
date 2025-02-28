package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	bill "github.com/vvvakho/feezy/workflows"
	billing "github.com/vvvakho/feezy/workflows"
	"go.temporal.io/sdk/client"
)

//TODO: add logger

type CreateBillRequest struct {
	UserID   string `json:"user_id"`
	Currency string `json:"currency"`
}

type CreateBillResponse struct {
	ID string `json:"id"`
}

// encore: api public method=POST path=/bills
func CreateBill(ctx context.Context, req *CreateBillRequest) (*CreateBillResponse, error) {
	//TODO: basic input validation before initializing client

	// Initialize client for Temporal connection
	c, err := client.Dial(client.Options{}) //TODO: add connection options
	if err != nil {
		return nil, fmt.Errorf("Error connecting to Temporal server: %v", err)
	}
	defer c.Close()

	// Generate a unique Bill ID
	billID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize bill ID: %v", err)
	}

	bill := bill.Bill{
		ID:       billID,
		Currency: req.Currency,
		UserID:   req.UserID,
	}

	// Start workflow asynchronously
	_, err = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        billID.String(), //TODO: may need to edit workflow id to not just be bill id
		TaskQueue: "create-bill-queue",
	}, billing.BillWorkflow, bill)

	if err != nil {
		return nil, fmt.Errorf("Unable to initiate workflow: %v", err)
	}

	return &CreateBillResponse{ID: billID.String()}, nil
}

type GetBillRequest struct {
	ID string `json:"user_id"`
}

type GetBillResponse struct {
	ID        string          `json:"id"`
	Items     []bill.LineItem `json:"items"`
	Total     bill.Money      `json:"total"`
	Currency  string          `json:"currency"`
	Status    bill.Status     `json:"status"`
	UserID    string          `json:"userId"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

//encore:api public method=GET path=/bills/:id
func GetBill(ctx context.Context, id string) (*GetBillResponse, error) {
	//TODO: check if bill active

	// Initialize client for Temporal connection
	c, err := client.Dial(client.Options{}) //TODO: add connection options
	if err != nil {
		return nil, fmt.Errorf("Error connecting to Temporal server: %v", err)
	}
	defer c.Close()

	// Query Temporal to check if workflow is active
	var billState billing.Bill //TODO: syntax...

	// Start signal synchronously
	resp, err := c.QueryWorkflow(ctx, id, "", "getBill")
	if err != nil {
		return nil, fmt.Errorf("Unable to initiate query signal: %v", err)
	}
	err = resp.Get(&billState)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse query response into Bill: %v", err)
	}

	return &GetBillResponse{
		ID:        billState.ID.String(),
		Items:     billState.Items,
		Total:     billState.Total,
		Currency:  billState.Currency,
		Status:    billState.Status,
		UserID:    billState.UserID,
		CreatedAt: billState.CreatedAt,
		UpdatedAt: billState.UpdatedAt,
	}, nil

	//TODO: logic if workflow no longer in Temporal
}

type CloseBillRequest struct {
	ID string `json:"user_id"`
}

type CloseBillResponse struct {
	Status string
}

//encore:api public method=POST path=/bills/:id
func CloseBill(ctx context.Context, id string) (*CloseBillResponse, error) {
	//TODO: check if bill active

	// Query Temporal to check if workflow is active

	// Initialize client for Temporal connection
	c, err := client.Dial(client.Options{}) //TODO: add connection options
	if err != nil {
		return nil, fmt.Errorf("Error connecting to Temporal server: %v", err)
	}
	defer c.Close()

	// closeSignal := bill.CloseBillSignal{
	//   Route: "closeBillSignal",
	// }

	err = c.SignalWorkflow(ctx, id, "", "closeBill", nil)
	if err != nil {
		return nil, fmt.Errorf("Error signaling CloseBill task: %v", err)
	}

	//TODO: logic if workflow no longer in Temporal
	return &CloseBillResponse{Status: "Success"}, nil
}
