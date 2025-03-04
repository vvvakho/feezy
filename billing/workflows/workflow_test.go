package workflows

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vvvakho/feezy/billing/service/domain"
	"go.temporal.io/sdk/testsuite"
)

// MockActivities defines a mock implementation for the workflow activities.
type MockActivities struct {
	mock.Mock
}

// Mock implementation of AddOpenBillToDB activity.
func (m *MockActivities) AddOpenBillToDB(ctx context.Context, bill *domain.Bill, requestID string) error {
	args := m.Called(ctx, bill, requestID)
	return args.Error(0)
}

// Mock implementation of AddClosedBillToDB activity.
func (m *MockActivities) AddClosedBillToDB(ctx context.Context, bill *domain.Bill, requestID string) error {
	args := m.Called(ctx, bill, requestID)
	return args.Error(0)
}

// UnitTestSuite defines the test suite for workflow tests.
type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env            *testsuite.TestWorkflowEnvironment
	mockActivities *MockActivities
}

// SetupTest sets up the test environment before each test.
func (s *UnitTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.mockActivities = new(MockActivities)
	s.env.RegisterActivity(s.mockActivities.AddOpenBillToDB)
	s.env.RegisterActivity(s.mockActivities.AddClosedBillToDB)
}

// AfterTest asserts that all expectations were met after each test.
func (s *UnitTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

// TestUnitTestSuite runs the test suite.
func TestUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

// TestBillWorkflow_Execution tests the execution of the BillWorkflow.
func (s *UnitTestSuite) TestBillWorkflow_Execution() {
	billID, _ := uuid.NewUUID()
	bill := &domain.Bill{
		ID:        billID,
		Status:    domain.BillOpen,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
		Total: domain.Money{
			Amount:   0,
			Currency: "USD",
		},
	}

	// Mock activities to execute successfully
	s.mockActivities.On("AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.mockActivities.On("AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Simulate signal arrival to prevent workflow from waiting indefinitely
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(AddLineItemRoute.Name, AddItemSignal{
			LineItem: domain.Item{
				ID:           uuid.New(),
				PricePerUnit: domain.Money{Amount: 20, Currency: "USD"},
				Quantity:     1,
			},
		})
	}, time.Millisecond*10)

	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(CloseBillRoute.Name, CloseBillSignal{
			Route:     "CloseBillRoute",
			RequestID: uuid.NewString(),
		})
	}, time.Millisecond*20)

	// Execute the workflow
	s.env.ExecuteWorkflow(BillWorkflow, bill)

	// Ensure the workflow has completed successfully
	require.True(s.T(), s.env.IsWorkflowCompleted())
	require.NoError(s.T(), s.env.GetWorkflowError())
}
