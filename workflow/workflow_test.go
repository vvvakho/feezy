package workflow_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"github.com/vvvakho/feezy/domain"
	"github.com/vvvakho/feezy/workflow"
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
		s.env.SignalWorkflow(workflow.AddLineItemRoute.Name, workflow.AddItemSignal{
			LineItem: domain.Item{
				ID:           uuid.New(),
				PricePerUnit: domain.Money{Amount: 20, Currency: "USD"},
				Quantity:     1,
			},
		})
	}, time.Millisecond*10)

	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(workflow.CloseBillRoute.Name, workflow.CloseBillSignal{
			Route:     "CloseBillRoute",
			RequestID: uuid.NewString(),
		})
	}, time.Millisecond*20)

	// Execute the workflow
	s.env.ExecuteWorkflow(workflow.BillWorkflow, bill)

	// Ensure the workflow has completed successfully
	require.True(s.T(), s.env.IsWorkflowCompleted())
	require.NoError(s.T(), s.env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_AddLineItem() {
	// Initialize a new bill
	bill := &domain.Bill{
		ID:     uuid.New(),
		Status: domain.BillOpen,
		Total: domain.Money{
			Amount:   0,
			Currency: "USD",
		},
		Items: []domain.Item{},
	}

	// Mock activities
	s.mockActivities.On("AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.mockActivities.On("AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil) // Add mock for closing

	// Register delayed callbacks to simulate signals
	s.env.RegisterDelayedCallback(func() {
		// Query the bill to ensure it's initially empty
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(0, len(queriedBill.Items))

		// Send a signal to add a line item
		item := domain.Item{
			ID:           uuid.New(),
			PricePerUnit: domain.Money{Amount: 50, Currency: "USD"},
			Quantity:     1,
		}
		s.env.SignalWorkflow(workflow.AddLineItemRoute.Name, workflow.AddItemSignal{
			LineItem: item,
		})
	}, time.Millisecond*1)

	s.env.RegisterDelayedCallback(func() {
		// Query after adding item
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(1, len(queriedBill.Items))
		s.Equal(domain.MinorUnit(50), queriedBill.Total.Amount)

		// Send signal to close the bill
		s.env.SignalWorkflow(workflow.CloseBillRoute.Name, workflow.CloseBillSignal{
			Route:     "CloseBillRoute",
			RequestID: uuid.NewString(),
		})
	}, time.Millisecond*10)

	// Execute the workflow
	s.env.ExecuteWorkflow(workflow.BillWorkflow, bill)

	// Ensure workflow completed successfully
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	// Verify mock calls
	s.mockActivities.AssertCalled(s.T(), "AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything)
	s.mockActivities.AssertCalled(s.T(), "AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything)
}

func (s *UnitTestSuite) Test_AddLineItemMultiple() {
	// Initialize a new bill
	bill := &domain.Bill{
		ID:     uuid.New(),
		Status: domain.BillOpen,
		Total: domain.Money{
			Amount:   0,
			Currency: "USD",
		},
		Items: []domain.Item{},
	}

	// Mock activities
	s.mockActivities.On("AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.mockActivities.On("AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil) // Add mock for closing

	itemID := uuid.New()

	// Register delayed callbacks to simulate signals
	s.env.RegisterDelayedCallback(func() {
		// Query the bill to ensure it's initially empty
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(0, len(queriedBill.Items))

		// Send a signal to add a line item
		item := domain.Item{
			ID:           itemID,
			PricePerUnit: domain.Money{Amount: 50, Currency: "USD"},
			Quantity:     1,
		}
		s.env.SignalWorkflow(workflow.AddLineItemRoute.Name, workflow.AddItemSignal{
			LineItem: item,
		})
	}, time.Millisecond*1)

	s.env.RegisterDelayedCallback(func() {
		time.Sleep(time.Millisecond * 5)

		// Query the bill to ensure first item was added
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(1, len(queriedBill.Items))

		// Send a signal to add 3 more of the same item
		item := domain.Item{
			ID:           itemID,
			PricePerUnit: domain.Money{Amount: 50, Currency: "USD"},
			Quantity:     3,
		}
		s.env.SignalWorkflow(workflow.AddLineItemRoute.Name, workflow.AddItemSignal{
			LineItem: item,
		})
	}, time.Millisecond*5)

	s.env.RegisterDelayedCallback(func() {
		time.Sleep(time.Millisecond * 10)

		// Query after adding multiple of the existing item
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(1, len(queriedBill.Items))               // len() represents unique items in bill
		s.Equal(int64(4), queriedBill.Items[0].Quantity) // here we check the quantity of a specific item
		s.Equal(domain.MinorUnit(200), queriedBill.Total.Amount)

		// Send signal to close the bill
		s.env.SignalWorkflow(workflow.CloseBillRoute.Name, workflow.CloseBillSignal{
			Route:     "CloseBillRoute",
			RequestID: uuid.NewString(),
		})
	}, time.Millisecond*15)

	// Execute the workflow
	s.env.ExecuteWorkflow(workflow.BillWorkflow, bill)

	// Ensure workflow completed successfully
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	// Verify mock calls
	s.mockActivities.AssertCalled(s.T(), "AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything)
	s.mockActivities.AssertCalled(s.T(), "AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything)
}

func (s *UnitTestSuite) Test_AddLineItemCurrencyDiff() {
	// Initialize a new bill
	bill := &domain.Bill{
		ID:     uuid.New(),
		Status: domain.BillOpen,
		Total: domain.Money{
			Amount:   0,
			Currency: "USD",
		},
		Items: []domain.Item{},
	}

	// Mock activities
	s.mockActivities.On("AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.mockActivities.On("AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil) // Add mock for closing

	Item := domain.Item{
		ID: uuid.New(),
		PricePerUnit: domain.Money{
			Amount:   275,
			Currency: "GEL",
		},
		Quantity: 1,
	}

	// Register delayed callbacks to simulate signals
	s.env.RegisterDelayedCallback(func() {
		// Query the bill to ensure it's initially empty
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(0, len(queriedBill.Items))

		// Send a signal to add a line item
		s.env.SignalWorkflow(workflow.AddLineItemRoute.Name, workflow.AddItemSignal{
			LineItem: Item,
		})
	}, time.Millisecond*1)

	s.env.RegisterDelayedCallback(func() {
		// Query after adding item
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(1, len(queriedBill.Items))
		// Check that total bill amount reflects converted currency of item price
		expAmount, err := domain.Convert(queriedBill.Total.Currency, Item.PricePerUnit.Currency, Item.PricePerUnit.Amount)
		s.NoError(err)
		s.Equal(expAmount, queriedBill.Total.Amount)

		// Send signal to close the bill
		s.env.SignalWorkflow(workflow.CloseBillRoute.Name, workflow.CloseBillSignal{
			Route:     "CloseBillRoute",
			RequestID: uuid.NewString(),
		})
	}, time.Millisecond*10)

	// Execute the workflow
	s.env.ExecuteWorkflow(workflow.BillWorkflow, bill)

	// Ensure workflow completed successfully
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	// Verify mock calls
	s.mockActivities.AssertCalled(s.T(), "AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything)
	s.mockActivities.AssertCalled(s.T(), "AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything)
}
