package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vvvakho/feezy/domain"
	"github.com/vvvakho/feezy/workflow"
)

// CreateBill creates a new bill for a given user and currency.
// It starts an asynchronous Temporal workflow to manage the bill lifecycle.
// Returns the newly created bill's ID, status, and metadata.
//
// encore: api public method=POST path=/bills
func (s *Service) CreateBill(ctx context.Context, req *CreateBillRequest) (*CreateBillResponse, error) {
	if err := validateCreateBillRequest(req); err != nil {
		return &CreateBillResponse{}, fmt.Errorf("Could not validate request: %v", err)
	}

	bill, err := domain.NewBill(req.UserID, req.Currency)
	if err != nil {
		return &CreateBillResponse{}, fmt.Errorf("Could not validate bill parameters: %v", err)
	}

	// Start workflow asynchronously
	err = createBillWorkflow(ctx, s.TemporalClient, bill)
	if err != nil {
		return &CreateBillResponse{}, fmt.Errorf("Could not create bill: %v", err)
	}

	return &CreateBillResponse{
		ID:        bill.ID.String(),
		UserID:    bill.UserID,
		Currency:  bill.Total.Currency,
		CreatedAt: bill.CreatedAt,
		Status:    string(bill.Status),
	}, nil
}

// GetBill retrieves the details of a specific bill by ID.
// If the bill is active, it queries the Temporal workflow for its current state.
// If the bill is closed, it fetches details from the database.
//
//encore:api public method=GET path=/bills/:id
func (s *Service) GetBill(ctx context.Context, id string) (*GetBillResponse, error) {
	// Check whether bill is active and in Temporal
	if err := isWorkflowRunning(s.TemporalClient, id); err != nil {
		// Check if bill is closed and in DB
		//TODO: s.DB.Query(ctx)
		return nil, fmt.Errorf("Bill not found or already closed: %v", err)
	}

	var bill domain.Bill

	if err := getBillQuery(ctx, s.TemporalClient, id, &bill); err != nil {
		return &GetBillResponse{}, fmt.Errorf("Unable to initiate bill query: %v", err)
	}

	return &GetBillResponse{
		ID:        bill.ID.String(),
		Items:     bill.Items,
		Total:     bill.Total,
		Status:    bill.Status,
		UserID:    bill.UserID,
		CreatedAt: bill.CreatedAt,
		UpdatedAt: bill.UpdatedAt,
	}, nil
}

// AddLineItemToBill adds a new line item to an active bill.
// If the bill is closed, the request is rejected.
// Sends an asynchronous signal to the Temporal workflow.
//
//encore:api public method=POST path=/bills/:id/items
func (s *Service) AddLineItemToBill(ctx context.Context, id string, req *AddLineItemRequest) (*AddLineItemResponse, error) {
	if err := validateAddLineItemRequest(req); err != nil {
		return &AddLineItemResponse{}, fmt.Errorf("Invalid request: %v", err)
	}

	if err := isWorkflowRunning(s.TemporalClient, id); err != nil {
		return nil, fmt.Errorf("Bill not found or already closed: %v", err)
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

	err = addLineItemSignal(ctx, s.TemporalClient, id, &billItem)
	if err != nil {
		return &AddLineItemResponse{}, fmt.Errorf("Unable to add line item to bill: %v", err)
	}

	return &AddLineItemResponse{Message: "ok"}, nil
}

//TODO: is PATCH appropriate ??

// RemoveLineItemFromBill removes an existing line item from an active bill.
// If the bill is closed, the request is rejected.
// Sends an asynchronous signal to the Temporal workflow.
//
//encore:api public method=PATCH path=/bills/:id/items
func (s *Service) RemoveLineItemFromBill(ctx context.Context, id string, req *RemoveLineItemRequest) (*RemoveLineItemResponse, error) {
	if err := validateRemoveLineItemRequest(req); err != nil {
		return &RemoveLineItemResponse{}, fmt.Errorf("Invalid request: %v", err)
	}

	if err := isWorkflowRunning(s.TemporalClient, id); err != nil {
		return nil, fmt.Errorf("Bill not found or already closed: %v", err)
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

	err = removeLineItemSignal(ctx, s.TemporalClient, id, &billItem)
	if err != nil {
		return &RemoveLineItemResponse{}, fmt.Errorf("Error signaling removeLineItem task: %v", err)
	}

	return &RemoveLineItemResponse{Message: "ok"}, nil
}

// CloseBill finalizes an open bill, preventing further modifications.
// Sends a signal to the Temporal workflow to mark the bill as closed.
// Closed bills are moved to the database for storage.
//
//encore:api public method=POST path=/bills/:id
func (s *Service) CloseBill(ctx context.Context, id string, req *CloseBillRequest) (*CloseBillResponse, error) {
	//TODO: check if bill active

	if err := validateCloseBillRequest(id, req); err != nil {
		return &CloseBillResponse{}, fmt.Errorf("Invalid request parameters: %v", err)
	}

	// Check if workflow is running
	if err := isWorkflowRunning(s.TemporalClient, id); err != nil {
		return nil, fmt.Errorf("Bill not found or already closed: %v", err)
	}

	err := closeBillSignal(ctx, s.TemporalClient, id, &workflow.CloseBillSignal{RequestID: req.RequestID})
	if err != nil {
		return nil, fmt.Errorf("Error signaling CloseBill task: %v", err)
	}

	return &CloseBillResponse{Status: "Success"}, nil //TODO: appropriate response?
}
