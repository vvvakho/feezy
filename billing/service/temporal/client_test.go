package temporalclient

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vvvakho/feezy/billing/service/domain"
	"github.com/vvvakho/feezy/billing/workflows"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
)

func TestInitTemporalClient(t *testing.T) {
	// Mock the TemporalDial function
	TemporalDial = func(options client.Options) (client.Client, error) {
		return &mocks.Client{}, nil
	}

	c, err := InitTemporalClient()
	assert.NoError(t, err)
	assert.NotNil(t, c)

	// Test error scenario
	TemporalDial = func(options client.Options) (client.Client, error) {
		return nil, fmt.Errorf("connection error")
	}

	c, err = InitTemporalClient()
	assert.Error(t, err)
	assert.Nil(t, c)
}

func TestIsWorkflowRunning(t *testing.T) {
	mockClient := &mocks.Client{}

	tests := []struct {
		name          string
		workflowID    string
		mockResponse  *workflowservice.DescribeWorkflowExecutionResponse
		mockError     error
		expectedError error
	}{
		{
			name:       "Workflow is running",
			workflowID: "running-workflow",
			mockResponse: &workflowservice.DescribeWorkflowExecutionResponse{
				WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
					Status: enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
				},
			},
			mockError:     nil,
			expectedError: nil,
		},
		{
			name:       "Workflow is not running",
			workflowID: "completed-workflow",
			mockResponse: &workflowservice.DescribeWorkflowExecutionResponse{
				WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
					Status: enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
				},
			},
			mockError:     nil,
			expectedError: errors.New("Workflow not running"),
		},
		{
			name:          "DescribeWorkflowExecution returns error",
			workflowID:    "unknown-workflow",
			mockResponse:  nil,
			mockError:     errors.New("workflow not found"),
			expectedError: errors.New("workflow not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.On("DescribeWorkflowExecution", mock.Anything, tt.workflowID, "").Return(tt.mockResponse, tt.mockError)

			err := IsWorkflowRunning(mockClient, tt.workflowID)
			assert.Equal(t, tt.expectedError, err)

			mockClient.AssertCalled(t, "DescribeWorkflowExecution", mock.Anything, tt.workflowID, "")
		})
	}
}

func TestAddLineItemSignal(t *testing.T) {
	mockClient := new(mocks.Client)
	ctx := context.Background()
	workflowID := "test-workflow-id"
	billItem := &domain.Item{}
	expectedSignal := workflows.AddItemSignal{LineItem: *billItem}

	mockClient.On("SignalWorkflow", ctx, workflowID, "", workflows.AddLineItemRoute.Name, expectedSignal).Return(nil)

	err := AddLineItemSignal(ctx, mockClient, workflowID, billItem)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestRemoveLineItemSignal(t *testing.T) {
	mockClient := new(mocks.Client)
	ctx := context.Background()
	workflowID := "test-workflow-id"
	billItem := &domain.Item{ /* initialize your item here */ }
	expectedSignal := workflows.AddItemSignal{LineItem: *billItem}

	mockClient.On("SignalWorkflow", ctx, workflowID, "", workflows.RemoveLineItemRoute.Name, expectedSignal).Return(nil)

	err := RemoveLineItemSignal(ctx, mockClient, workflowID, billItem)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestCloseBillSignal(t *testing.T) {
	mockClient := new(mocks.Client)
	ctx := context.Background()
	workflowID := "test-workflow-id"
	closeReq := &workflows.CloseBillSignal{ /* initialize your request here */ }

	mockClient.On("SignalWorkflow", ctx, workflowID, "", workflows.CloseBillRoute.Name, closeReq).Return(nil)

	err := CloseBillSignal(ctx, mockClient, workflowID, closeReq)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestCloseBillUpdate(t *testing.T) {
	mockClient := new(mocks.Client)
	ctx := context.Background()
	workflowID := "test-workflow-id"
	closeReq := &workflows.CloseBillSignal{RequestID: "test-request-id"}
	expectedBill := &domain.Bill{ /* initialize your bill here */ }

	mockUpdateHandle := new(mocks.WorkflowUpdateHandle)
	mockUpdateHandle.On("Get", ctx, mock.Anything).Run(func(args mock.Arguments) {
		*args.Get(1).(**domain.Bill) = expectedBill
	}).Return(nil)

	mockClient.On("UpdateWorkflow", ctx, mock.Anything).Return(mockUpdateHandle, nil)
	mockClient.On("SignalWorkflow", ctx, workflowID, "", workflows.CloseWorkflowRoute.Name, closeReq).Return(nil)

	result, err := CloseBillUpdate(ctx, mockClient, workflowID, closeReq)

	assert.NoError(t, err)
	assert.Equal(t, expectedBill, result)
	mockClient.AssertExpectations(t)
	mockUpdateHandle.AssertExpectations(t)
}

func TestCloseWorkflowSignal(t *testing.T) {
	mockClient := new(mocks.Client)
	ctx := context.Background()
	workflowID := "test-workflow-id"
	closeReq := &workflows.CloseBillSignal{ /* initialize your request here */ }

	mockClient.On("SignalWorkflow", ctx, workflowID, "", workflows.CloseWorkflowRoute.Name, closeReq).Return(nil)

	err := CloseWorkflowSignal(ctx, mockClient, workflowID, closeReq)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}
