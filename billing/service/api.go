package billing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vvvakho/feezy/billing/service/domain"
	"github.com/vvvakho/feezy/billing/workflows"
)

// CreateBill creates a new bill for a given user and currency.
// It starts an asynchronous Temporal workflows to manage the bill lifecycle.
// Returns the newly created bill's ID, status, and metadata.
//
// encore: api private method=POST path=/bills
func (s *Service) CreateBill(ctx context.Context, req *CreateBillRequest) (*CreateBillResponse, error) {
	if err := validateCreateBillRequest(req); err != nil {
		return nil, fmt.Errorf("Could not validate request: %v", err)
	}

	bill, err := domain.NewBill(req.UserID, req.Currency)
	if err != nil {
		return nil, fmt.Errorf("Could not validate bill parameters: %v", err)
	}

	// Start workflows asynchronously
	err = s.Execution.CreateBillWorkflow(ctx, bill)
	if err != nil {
		return nil, fmt.Errorf("Could not create bill: %v", err)
	}

	return &CreateBillResponse{
		ID:        bill.ID.String(),
		UserID:    bill.UserID.String(),
		Currency:  bill.Total.Currency,
		CreatedAt: bill.CreatedAt,
		Status:    string(bill.Status),
	}, nil
}

// GetBill retrieves the details of a specific bill by ID.
// If the bill is active, it queries the Temporal workflows for its current state.
// If the bill is closed, it fetches details from the database.
//
//encore:api private method=GET path=/bills/:id
func (s *Service) GetBill(ctx context.Context, id string) (*GetBillResponse, error) {
	// Check cache for bill entry
	cachedBill, err := s.Cache.Get(ctx, id)
	if err == nil {
		// Cache hit; return the cached bill
		fmt.Printf("Cache HIT!")
		return &cachedBill, nil
	}

	// Cache miss, proceed with bill retrieval from DB
	// Check if bill exists in open_bills DB
	_, err = s.Repository.GetOpenBillFromDB(ctx, id)
	if err == nil {
		// Check if the workflows is running (only if bill is open)
		if err := s.Execution.IsWorkflowRunning(id); err != nil {
			return nil, fmt.Errorf("Unexpected error fetching bill: %v", err)
		} else {
			// Query Temporal Workflow for Bill Details
			var bill domain.Bill
			if err := s.Execution.GetBillQuery(ctx, id, &bill); err != nil {
				return nil, fmt.Errorf("Unable to query bill from Temporal: %v", err)
			}

			resp := GetBillResponse{
				ID:        bill.ID.String(),
				Items:     bill.Items,
				Total:     bill.Total,
				Status:    bill.Status,
				UserID:    bill.UserID.String(),
				CreatedAt: bill.CreatedAt,
				UpdatedAt: bill.UpdatedAt,
			}

			// Update the cache
			if err := s.Cache.Set(ctx, id, resp); err != nil {
				return nil, fmt.Errorf("failed to update cache: %v", err)
			}

			return &resp, nil
		}
	}

	// If bill is not in open_bills, check closed_bills DB
	closedBill, err := s.Repository.GetClosedBillFromDB(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Bill not found: %v", err)
	}

	// Get bill items from closed_bill_items
	closedBillItems, err := s.Repository.GetClosedBillItemsFromDB(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Bill items not found: %v", err)
	}

	resp := GetBillResponse{
		ID:        closedBill.ID.String(),
		Items:     closedBillItems,
		Total:     closedBill.Total,
		Status:    closedBill.Status,
		UserID:    closedBill.UserID.String(),
		CreatedAt: closedBill.CreatedAt,
		UpdatedAt: closedBill.UpdatedAt,
	}

	// Update Cache
	if err := s.Cache.Set(ctx, id, resp); err != nil {
		return nil, fmt.Errorf("failed to update cache: %v", err)
	}

	return &resp, nil
}

// AddLineItemToBill adds a new line item to an active bill.
// If the bill is closed, the request is rejected.
// Sends an asynchronous signal to the Temporal workflows.
//
//encore:api private method=POST path=/bills/:id/items
func (s *Service) AddLineItemToBill(ctx context.Context, id string, req *AddLineItemRequest) (*AddLineItemResponse, error) {
	if err := validateAddLineItemRequest(req); err != nil {
		return nil, fmt.Errorf("Invalid request: %v", err)
	}

	// Check if bill exists in open_bills DB
	_, err := s.Repository.GetOpenBillFromDB(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Bill not found or already closed: %v", err)
	}

	// Check if workflow is running
	if err := s.Execution.IsWorkflowRunning(id); err != nil {
		return nil, fmt.Errorf("Unexpected error fetching bill: %v", err)
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

	err = s.Execution.AddLineItemSignal(ctx, id, &billItem)
	if err != nil {
		return nil, fmt.Errorf("Unable to add line item to bill: %v", err)
	}

	return &AddLineItemResponse{Message: "Request has been sent"}, nil
}

// RemoveLineItemFromBill removes an existing line item from an active bill.
// If the bill is closed, the request is rejected.
// Sends an asynchronous signal to the Temporal workflows.
//
//encore:api private method=PATCH path=/bills/:id/items
func (s *Service) RemoveLineItemFromBill(ctx context.Context, id string, req *RemoveLineItemRequest) (*RemoveLineItemResponse, error) {
	if err := validateRemoveLineItemRequest(req); err != nil {
		return nil, fmt.Errorf("Invalid request: %v", err)
	}

	// Check if bill exists in open_bills DB
	_, err := s.Repository.GetOpenBillFromDB(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Bill not found or already closed: %v", err)
	}

	// Check if workflow is running
	if err := s.Execution.IsWorkflowRunning(id); err != nil {
		return nil, fmt.Errorf("Unexpected error fetching bill: %v", err)
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

	err = s.Execution.RemoveLineItemSignal(ctx, id, &billItem)
	if err != nil {
		return nil, fmt.Errorf("Error signaling removeLineItem task: %v", err)
	}

	return &RemoveLineItemResponse{Message: "Request has been sent"}, nil
}

// CloseBill finalizes an open bill, preventing further modifications.
// Sends a signal to the Temporal workflows to mark the bill as closed.
// Closed bills are moved to the database for storage.
//
//encore:api private method=PATCH path=/bills/:id
func (s *Service) CloseBill(ctx context.Context, id string, req *CloseBillRequest) (*CloseBillResponse, error) {
	if err := validateCloseBillRequest(id, req); err != nil {
		return nil, fmt.Errorf("Invalid request parameters: %v", err)
	}

	// Check if bill exists in open_bills DB
	_, err := s.Repository.GetOpenBillFromDB(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Bill not found or already closed: %v", err)
	}

	// Check if workflow is running
	if err := s.Execution.IsWorkflowRunning(id); err != nil {
		return nil, fmt.Errorf("Unexpected error fetching bill: %v", err)
	}

	// Perform a synchronous request to close bill and return its state
	// Alternatively, we have an option to use CloseBillSignal() for asynchronicity
	closedBill, err := s.Execution.CloseBillUpdate(ctx, id, &workflows.CloseBillSignal{RequestID: req.RequestID})
	if err != nil {
		return nil, fmt.Errorf("Error sending CloseBill update: %v", err)
	}

	return &CloseBillResponse{Bill: closedBill, Status: "Bill successfully closed"}, nil
}
