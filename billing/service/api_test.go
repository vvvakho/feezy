package billing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vvvakho/feezy/billing/service/domain"
	"go.temporal.io/sdk/mocks"
)

func (m *MockTemporalClient) CreateBillWorkflow(ctx context.Context, client interface{}, bill *domain.Bill) error {
	args := m.Called(ctx, client, bill)
	return args.Error(0)
}

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetOpenBillFromDB(ctx context.Context, billID string) (*domain.Bill, error) {
	args := m.Called(ctx, billID)
	return args.Get(0).(*domain.Bill), args.Error(1)
}

func (m *MockRepository) GetClosedBillFromDB(ctx context.Context, billID string) (*domain.Bill, error) {
	args := m.Called(ctx, billID)
	return args.Get(0).(*domain.Bill), args.Error(1)
}

func (m *MockRepository) GetClosedBillItemsFromDB(ctx context.Context, billID string) ([]domain.Item, error) {
	args := m.Called(ctx, billID)
	return args.Get(0).([]domain.Item), args.Error(1)
}

func TestCreateBill_Success(t *testing.T) {
	mockTemporalClient := new(mocks.Client)
	mockRepo := new(MockRepository)

	// Mock service with dependencies
	service := &Service{
		TemporalClient: mockTemporalClient,
		Repository:     mockRepo,
	}

	// Sample request
	req := &CreateBillRequest{
		UserID:   uuid.New().String(),
		Currency: "USD",
	}

	userID, err := uuid.Parse(req.UserID)
	assert.NoError(t, err)

	// Mock bill creation logic
	expectedBill := &domain.Bill{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: time.Now(),
		Total:     domain.Money{Currency: "USD"},
		Status:    domain.BillOpen,
	}

	// Mock external function behavior
	mockTemporalClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&mocks.WorkflowRun{}, nil)

	resp, err := service.CreateBill(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedBill.UserID.String(), resp.UserID)
	assert.Equal(t, expectedBill.Total.Currency, resp.Currency)
	assert.Equal(t, string(expectedBill.Status), resp.Status)
	mockTemporalClient.AssertExpectations(t)
}

func TestCreateBill_ValidationError(t *testing.T) {
	mockTemporalClient := new(mocks.Client)
	mockRepo := new(MockRepository)

	// Mock service with dependencies
	service := &Service{
		TemporalClient: mockTemporalClient,
		Repository:     mockRepo,
	}

	tests := []struct {
		name   string
		req    *CreateBillRequest
		errMsg string
	}{
		{
			name:   "Empty UserID",
			req:    &CreateBillRequest{UserID: "", Currency: "USD"},
			errMsg: "Could not validate request",
		},
		{
			name:   "Invalid Currency",
			req:    &CreateBillRequest{UserID: uuid.New().String(), Currency: "INVALID"},
			errMsg: "Could not validate request",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := service.CreateBill(context.Background(), tc.req)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errMsg)
			assert.Nil(t, resp)
		})
	}
}

func TestCreateBill_WorkflowError(t *testing.T) {
	mockTemporalClient := new(mocks.Client)
	mockRepo := new(MockRepository)

	service := &Service{
		TemporalClient: mockTemporalClient,
		Repository:     mockRepo,
	}

	req := &CreateBillRequest{
		UserID:   uuid.New().String(),
		Currency: "USD",
	}

	expectedErr := "Temporal workflow execution failed"

	mockTemporalClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return((*mocks.WorkflowRun)(nil), fmt.Errorf(expectedErr))

	resp, err := service.CreateBill(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Could not create bill")
	assert.Contains(t, err.Error(), expectedErr)
	assert.Nil(t, resp)

	mockTemporalClient.AssertExpectations(t)
}

func TestCreateBill_TemporalCalledWithCorrectParams(t *testing.T) {
	mockTemporalClient := new(mocks.Client)
	mockRepo := new(MockRepository)

	service := &Service{
		TemporalClient: mockTemporalClient,
		Repository:     mockRepo,
	}

	req := &CreateBillRequest{
		UserID:   uuid.New().String(),
		Currency: "USD",
	}

	userID, err := uuid.Parse(req.UserID)
	assert.NoError(t, err)

	expectedBill := &domain.Bill{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: time.Now(),
		Total:     domain.Money{Currency: "USD"},
		Status:    domain.BillOpen,
	}

	mockTemporalClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&mocks.WorkflowRun{}, nil).
		Run(func(args mock.Arguments) {
			// Extract the bill argument from ExecuteWorkflow
			billArg, ok := args.Get(3).(*domain.Bill)
			assert.True(t, ok)
			assert.Equal(t, expectedBill.Total.Currency, billArg.Total.Currency)
			assert.Equal(t, expectedBill.UserID, billArg.UserID)
		})

	resp, err := service.CreateBill(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockTemporalClient.AssertExpectations(t)
}
