package api

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	bill "github.com/vvvakho/feezy/workflows"
	billing "github.com/vvvakho/feezy/workflows"
	"go.temporal.io/api/enums/v1"
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
	ID        string      `json:"id"`
	Items     []bill.Item `json:"items"`
	Total     bill.Money  `json:"total"`
	Currency  string      `json:"currency"`
	Status    bill.Status `json:"status"`
	UserID    string      `json:"userId"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
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

type AddLineItemToBillRequest struct {
	ID           string
	Quantity     int64
	Description  string
	PricePerUnit billing.Money
}

type AddLineItemToBillResponse struct {
	Message string
}

//encore:api public method=POST path=/bills/:id/items
func AddLineItemToBill(ctx context.Context, id string, req AddLineItemToBillRequest) (*AddLineItemToBillResponse, error) {
	// Initialize Temporal client
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil, fmt.Errorf("Error connecting to Temporal server: %v", err) //TODO: refactor to custom error
	}
	defer c.Close()

	// Check if bill exists and is active
	ok, err := isWorkflowRunning(c, id)
	if !ok {
		return nil, fmt.Errorf("Bill not found or already closed: %v", err) //TODO: refactor to custom error
	}

	itemID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, fmt.Errorf("Invalid ID: %v", err)
	}

	item := billing.Item{
		ID:           itemID,
		Quantity:     req.Quantity,
		Description:  req.Description,
		PricePerUnit: req.PricePerUnit,
	}

	err = c.SignalWorkflow(ctx, id, "", "addLineItem", bill.AddItemSignal{LineItem: item})
	if err != nil {
		return nil, fmt.Errorf("Error signaling addLineItem task: %v", err)
	}

	return &AddLineItemToBillResponse{Message: "ok"}, nil
}

type RemoveLineItemToBillRequest struct {
	ID           string
	Quantity     int64
	Description  string
	PricePerUnit billing.Money
}

type RemoveLineItemToBillResponse struct {
	Message string
}

//TODO: is PATCH appropriate ??

//encore:api public method=PATCH path=/bills/:id/items
func RemoveLineItemToBill(ctx context.Context, id string, req RemoveLineItemToBillRequest) (*RemoveLineItemToBillResponse, error) {
	// Initialize Temporal client
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil, fmt.Errorf("Error connecting to Temporal server: %v", err) //TODO: refactor to custom error
	}
	defer c.Close()

	// Check if bill exists and is active
	ok, err := isWorkflowRunning(c, id)
	if !ok {
		return nil, fmt.Errorf("Bill not found or already closed: %v", err) //TODO: refactor to custom error
	}

	itemID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, fmt.Errorf("Invalid ID: %v", err)
	}

	item := billing.Item{
		ID:           itemID,
		Quantity:     req.Quantity,
		Description:  req.Description,
		PricePerUnit: req.PricePerUnit,
	}

	err = c.SignalWorkflow(ctx, id, "", "removeLineItem", bill.AddItemSignal{LineItem: item})
	if err != nil {
		return nil, fmt.Errorf("Error signaling removeLineItem task: %v", err)
	}

	return &RemoveLineItemToBillResponse{Message: "ok"}, nil
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

	// Connect to Temporal
	c, err := client.Dial(client.Options{}) //TODO: add connection options
	if err != nil {
		return nil, fmt.Errorf("Error connecting to Temporal server: %v", err)
	}
	defer c.Close()

	// Check if workflow is running
	ok, err := isWorkflowRunning(c, id)
	if !ok {
		return nil, fmt.Errorf("Bill not found or already closed: %v", err)
	}

	// closeSignal := bill.CloseBillSignal{
	//   Route: "closeBillSignal",
	// }

	err = c.SignalWorkflow(ctx, id, "", "closeBill", nil)
	if err != nil {
		return nil, fmt.Errorf("Error signaling CloseBill task: %v", err)
	}

	//TODO: logic if workflow no longer in Temporal

	return &CloseBillResponse{Status: "Success"}, nil //TODO: appropriate response?
}

func isWorkflowRunning(c client.Client, workflowID string) (bool, error) {
	response, err := c.DescribeWorkflowExecution(context.Background(), workflowID, "")
	if err != nil {
		return false, err
	}
	return response.WorkflowExecutionInfo.Status == enums.WORKFLOW_EXECUTION_STATUS_RUNNING, nil
}
