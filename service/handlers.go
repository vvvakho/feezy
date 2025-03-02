package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vvvakho/feezy/domain"
	"github.com/vvvakho/feezy/workflow"
	"go.temporal.io/sdk/client"
)

// encore: api public method=POST path=/bills
func (s *Service) CreateBill(ctx context.Context, req *CreateBillRequest) (*CreateBillResponse, error) {
	//TODO: basic input validation before initializing client
	// check if user exists

	_, err := domain.IsValidCurrency(req.Currency)
	if err != nil {
		return nil, err
	}

	// Generate a unique Bill ID
	billID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize bill ID: %v", err)
	}

	bill := domain.Bill{
		ID:     billID,
		UserID: req.UserID,
		Items:  []domain.Item{},
		Total:  domain.Money{Amount: 0, Currency: req.Currency},
		Status: domain.BillOpen,
	}

	// Start workflow asynchronously
	c := *s.TemporalClient
	_, err = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        billID.String(), //TODO: may need to edit workflow id to not just be bill id
		TaskQueue: "create-bill-queue",
	}, workflow.BillWorkflow, bill)

	if err != nil {
		return nil, fmt.Errorf("Unable to initiate workflow: %v", err)
	}

	return &CreateBillResponse{ID: billID.String()}, nil
}

//encore:api public method=GET path=/bills/:id
func (s *Service) GetBill(ctx context.Context, id string) (*GetBillResponse, error) {
	//TODO: check if bill active
	// do we first check db for closed bill or do we first try temporal?
	// we'll be removing records from temporal after they complete
	// so maybe check temporal, then its status, and if not present then check db

	var billState domain.Bill //TODO: syntax...

	// Start signal synchronously
	c := *s.TemporalClient
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
		Status:    billState.Status,
		UserID:    billState.UserID,
		CreatedAt: billState.CreatedAt,
		UpdatedAt: billState.UpdatedAt,
	}, nil

	//TODO: logic if workflow no longer in Temporal
}

//encore:api public method=POST path=/bills/:id/items
func (s *Service) AddLineItemToBill(ctx context.Context, id string, req AddLineItemToBillRequest) (*AddLineItemToBillResponse, error) {
	// Validate input
	if req.Quantity < 1 {
		return &AddLineItemToBillResponse{}, fmt.Errorf("Invalid item quantity: %v", req.Quantity)
	}
	_, err := domain.IsValidCurrency(req.PricePerUnit.Currency)
	if err != nil {
		return &AddLineItemToBillResponse{}, err
	}

	// Initialize Temporal client
	c := *s.TemporalClient

	// Check if bill exists and is active
	ok, err := isWorkflowRunning(c, id)
	if !ok {
		return nil, fmt.Errorf("Bill not found or already closed: %v", err) //TODO: refactor to custom error
	}

	itemID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, fmt.Errorf("Invalid ID: %v", err)
	}

	billItem := domain.Item{
		ID:           itemID,
		Quantity:     req.Quantity,
		Description:  req.Description,
		PricePerUnit: req.PricePerUnit,
	}

	err = c.SignalWorkflow(ctx, id, "", "addLineItem", workflow.AddItemSignal{LineItem: billItem})
	if err != nil {
		return nil, fmt.Errorf("Error signaling addLineItem task: %v", err)
	}

	return &AddLineItemToBillResponse{Message: "ok"}, nil
}

//TODO: BatchAddLineItems

//TODO: is PATCH appropriate ??

//encore:api public method=PATCH path=/bills/:id/items
func (s *Service) RemoveLineItemToBill(ctx context.Context, id string, req RemoveLineItemFromBillRequest) (*RemoveLineItemFromBillResponse, error) {
	// Initialize Temporal client
	c := *s.TemporalClient

	// Check if bill exists and is active
	ok, err := isWorkflowRunning(c, id)
	if !ok {
		return nil, fmt.Errorf("Bill not found or already closed: %v", err) //TODO: refactor to custom error
	}

	itemID, err := uuid.Parse(req.ID)
	if err != nil {
		return nil, fmt.Errorf("Invalid ID: %v", err)
	}

	billItem := domain.Item{
		ID:           itemID,
		Quantity:     req.Quantity,
		Description:  req.Description,
		PricePerUnit: req.PricePerUnit,
	}

	err = c.SignalWorkflow(ctx, id, "", "removeLineItem", workflow.AddItemSignal{LineItem: billItem})
	if err != nil {
		return nil, fmt.Errorf("Error signaling removeLineItem task: %v", err)
	}

	return &RemoveLineItemFromBillResponse{Message: "ok"}, nil
}

//encore:api public method=POST path=/bills/:id
func (s *Service) CloseBill(ctx context.Context, id string, req CloseBillRequest) (*CloseBillResponse, error) {
	//TODO: check if bill active

	// Query Temporal to check if workflow is active

	// Connect to Temporal
	c := *s.TemporalClient

	// Check if workflow is running
	ok, err := isWorkflowRunning(c, id)
	if !ok {
		return nil, fmt.Errorf("Bill not found or already closed: %v", err)
	}

	// closeSignal := billing.CloseBillSignal{
	//   Route: "closeBillSignal",
	// }

	if req.RequestID == "" {
		req.RequestID = uuid.NewString()
	}

	err = c.SignalWorkflow(ctx, id, "", "closeBill", workflow.CloseBillSignal{RequestID: req.RequestID})
	if err != nil {
		return nil, fmt.Errorf("Error signaling CloseBill task: %v", err)
	}

	//TODO: logic if workflow no longer in Temporal

	return &CloseBillResponse{Status: "Success"}, nil //TODO: appropriate response?
}
