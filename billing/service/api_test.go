package billing

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	mock_billing "github.com/vvvakho/feezy/billing/mocks"
	"github.com/vvvakho/feezy/billing/service/domain"
	"go.uber.org/mock/gomock"
)

func TestCreateBill(t *testing.T) {
	// Define test cases
	tests := []struct {
		name                string
		userID              string
		currency            string
		mockError           error
		expectError         bool
		shouldCallExecution bool
	}{
		{
			name:                "Success Case",
			userID:              uuid.New().String(),
			currency:            "USD",
			mockError:           nil,
			expectError:         false,
			shouldCallExecution: true,
		},
		{
			name:                "Failure Case - Workflow Error",
			userID:              uuid.New().String(),
			currency:            "USD",
			mockError:           assert.AnError,
			expectError:         true,
			shouldCallExecution: true,
		},
		{
			name:                "Failure Case - Invalid UserID",
			userID:              "invalid-uuid",
			currency:            "USD",
			mockError:           nil,
			expectError:         true,
			shouldCallExecution: false,
		},
		{
			name:                "Failure Case - Invalid Currency",
			userID:              uuid.New().String(),
			currency:            "XYZ",
			mockError:           nil,
			expectError:         true,
			shouldCallExecution: false,
		},
		{
			name:                "Failure Case - Empty UserID",
			userID:              "",
			currency:            "USD",
			mockError:           nil,
			expectError:         true,
			shouldCallExecution: false,
		},
		{
			name:                "Failure Case - Empty Currency",
			userID:              uuid.New().String(),
			currency:            "",
			mockError:           nil,
			expectError:         true,
			shouldCallExecution: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize GoMock controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock dependencies
			mockExecution := mock_billing.NewMockExecution(ctrl)
			mockRepository := mock_billing.NewMockRepository(ctrl)

			// Initialize the service with mocks
			s := &Service{
				Execution:  mockExecution,
				Repository: mockRepository,
			}

			// Define test input
			req := &CreateBillRequest{
				UserID:   tt.userID,
				Currency: tt.currency,
			}

			// Set expectations using GoMock only if execution should be called
			if tt.shouldCallExecution {
				mockExecution.EXPECT().
					CreateBillWorkflow(gomock.Any(), gomock.Any()).
					Return(tt.mockError).
					Times(1)
			}

			// Call the function with debug logging
			fmt.Println("Running test case:", tt.name)
			resp, err := s.CreateBill(context.Background(), req)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestGetBill(t *testing.T) {
	tests := []struct {
		name             string
		billID           string
		openBillExists   bool
		workflowRunning  bool
		queryError       error
		closedBillExists bool
		closedBillError  error
		expectError      bool
	}{
		{
			name:             "Success - Open Bill with Workflow",
			billID:           uuid.New().String(),
			openBillExists:   true,
			workflowRunning:  true,
			queryError:       nil,
			closedBillExists: false,
			expectError:      false,
		},
		{
			name:             "Success - Closed Bill",
			billID:           uuid.New().String(),
			openBillExists:   false,
			closedBillExists: true,
			closedBillError:  nil,
			expectError:      false,
		},
		{
			name:             "Failure - Bill Not Found",
			billID:           uuid.New().String(),
			openBillExists:   false,
			closedBillExists: false,
			expectError:      true,
		},
		{
			name:            "Failure - Workflow Query Error",
			billID:          uuid.New().String(),
			openBillExists:  true,
			workflowRunning: true,
			queryError:      assert.AnError,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockExecution := mock_billing.NewMockExecution(ctrl)
			mockRepository := mock_billing.NewMockRepository(ctrl)

			s := &Service{
				Execution:  mockExecution,
				Repository: mockRepository,
			}

			ctx := context.Background()

			// Mock open bill retrieval
			if tt.openBillExists {
				mockRepository.EXPECT().GetOpenBillFromDB(ctx, tt.billID).Return(&domain.Bill{}, nil)
				mockExecution.EXPECT().IsWorkflowRunning(tt.billID).Return(nil)

				if tt.workflowRunning {
					mockExecution.EXPECT().GetBillQuery(ctx, tt.billID, gomock.Any()).Return(tt.queryError)
				}
			} else {
				mockRepository.EXPECT().GetOpenBillFromDB(ctx, tt.billID).Return(nil, assert.AnError)
			}

			// Mock closed bill retrieval
			if !tt.openBillExists && tt.closedBillExists {
				mockRepository.EXPECT().GetClosedBillFromDB(ctx, tt.billID).Return(&domain.Bill{}, tt.closedBillError)
				mockRepository.EXPECT().GetClosedBillItemsFromDB(ctx, tt.billID).Return([]domain.Item{}, nil)
			} else if !tt.openBillExists {
				mockRepository.EXPECT().GetClosedBillFromDB(ctx, tt.billID).Return(nil, assert.AnError)
			}

			resp, err := s.GetBill(ctx, tt.billID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

func TestAddLineItemToBill(t *testing.T) {
	tests := []struct {
		name            string
		billID          string
		request         *AddLineItemRequest
		openBillExists  bool
		workflowRunning bool
		mockError       error
		expectError     bool
		shouldValidate  bool
	}{
		{
			name:   "Success - Line Item Added",
			billID: uuid.New().String(),
			request: &AddLineItemRequest{
				ID:          uuid.New().String(),
				Quantity:    2,
				Description: "Test Item",
				PricePerUnit: domain.Money{
					Amount:   10,
					Currency: "USD",
				},
			},
			openBillExists:  true,
			workflowRunning: true,
			mockError:       nil,
			expectError:     false,
			shouldValidate:  true,
		},
		{
			name:   "Failure - Bill Not Found",
			billID: uuid.New().String(),
			request: &AddLineItemRequest{
				ID:          uuid.New().String(),
				Quantity:    1,
				Description: "Invalid Bill",
				PricePerUnit: domain.Money{
					Amount:   5,
					Currency: "USD",
				},
			},
			openBillExists: false,
			mockError:      nil,
			expectError:    true,
			shouldValidate: true,
		},
		{
			name:   "Failure - Invalid Item ID",
			billID: uuid.New().String(),
			request: &AddLineItemRequest{
				ID:          "invalid-uuid",
				Quantity:    1,
				Description: "Test Item",
				PricePerUnit: domain.Money{
					Amount:   10,
					Currency: "USD",
				},
			},
			openBillExists: true,
			mockError:      nil,
			expectError:    true,
			shouldValidate: false,
		},
		{
			name:   "Failure - Negative Price",
			billID: uuid.New().String(),
			request: &AddLineItemRequest{
				ID:          uuid.New().String(),
				Quantity:    1,
				Description: "Negative Price Item",
				PricePerUnit: domain.Money{
					Amount:   -5,
					Currency: "USD",
				},
			},
			openBillExists: true,
			mockError:      nil,
			expectError:    true,
			shouldValidate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockExecution := mock_billing.NewMockExecution(ctrl)
			mockRepository := mock_billing.NewMockRepository(ctrl)

			s := &Service{
				Execution:  mockExecution,
				Repository: mockRepository,
			}

			ctx := context.Background()

			// Skip mock expectations if validation should fail first
			if tt.shouldValidate {
				// Mock bill retrieval
				if tt.openBillExists {
					mockRepository.EXPECT().GetOpenBillFromDB(ctx, tt.billID).Return(&domain.Bill{}, nil)
					mockExecution.EXPECT().IsWorkflowRunning(tt.billID).Return(nil)
				} else {
					mockRepository.EXPECT().GetOpenBillFromDB(ctx, tt.billID).Return(nil, assert.AnError)
				}

				// Mock adding line item
				if tt.openBillExists && !tt.expectError {
					mockExecution.EXPECT().AddLineItemSignal(ctx, tt.billID, gomock.Any()).Return(tt.mockError)
				}
			}

			resp, err := s.AddLineItemToBill(ctx, tt.billID, tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}
