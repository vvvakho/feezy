package execution

import (
	"context"
	"fmt"

	"github.com/vvvakho/feezy/billing/conf"
	"github.com/vvvakho/feezy/billing/service/domain"
	"github.com/vvvakho/feezy/billing/workflows"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

type TemporalClient struct {
	Client client.Client
}

// Dependency injection -- primarily for mock testing
var WorkflowDial = client.Dial

func New() (*TemporalClient, error) {
	// Connect to Temporal
	c, err := WorkflowDial(conf.TEMPORAL_CLIENT_CONF)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to Temporal: %v", err)
	}
	return &TemporalClient{Client: c}, nil
}

func (tc *TemporalClient) Close() {
	tc.Client.Close()
}

func (tc *TemporalClient) IsWorkflowRunning(workflowsID string) error {
	response, err := tc.Client.DescribeWorkflowExecution(context.Background(), workflowsID, "")
	if err != nil {
		return err
	}
	if response.WorkflowExecutionInfo.Status != enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return fmt.Errorf("Workflow not running")
	}
	return nil
}

func (tc *TemporalClient) GetBillQuery(ctx context.Context, w string, bill *domain.Bill) error {
	// Start signal synchronously
	resp, err := tc.Client.QueryWorkflow(ctx, w, "", "getBill")
	if err != nil {
		return fmt.Errorf("Unable to initiate query signal: %v", err)
	}
	err = resp.Get(&bill)
	if err != nil {
		return fmt.Errorf("Unable to parse query response into Bill: %v", err)
	}

	return nil
}

func (tc *TemporalClient) CreateBillWorkflow(ctx context.Context, bill *domain.Bill) error {
	_, err := tc.Client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        bill.ID.String(),
		TaskQueue: "create-bill-queue",
	}, workflows.BillWorkflow, bill)

	if err != nil {
		return fmt.Errorf("Unable to initiate workflows: %v", err)
	}

	return nil
}

func (tc *TemporalClient) AddLineItemSignal(ctx context.Context, w string, billItem *domain.Item) error {
	err := tc.Client.SignalWorkflow(ctx, w, "", workflows.AddLineItemRoute.Name, workflows.AddItemSignal{LineItem: *billItem})
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflows.AddLineItemRoute.Name, err)
	}

	return nil
}

func (tc *TemporalClient) RemoveLineItemSignal(ctx context.Context, w string, billItem *domain.Item) error {
	err := tc.Client.SignalWorkflow(ctx, w, "", workflows.RemoveLineItemRoute.Name, workflows.AddItemSignal{LineItem: *billItem})
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflows.RemoveLineItemRoute.Name, err)
	}

	return nil
}

func (tc *TemporalClient) CloseBillSignal(ctx context.Context, w string, closeReq *workflows.CloseBillSignal) error {
	err := tc.Client.SignalWorkflow(ctx, w, "", workflows.CloseBillRoute.Name, closeReq)
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflows.CloseBillRoute.Name, err)
	}

	return nil
}

func (tc *TemporalClient) CloseBillUpdate(ctx context.Context, w string, closeReq *workflows.CloseBillSignal) (*domain.Bill, error) {
	updateHandle, err := tc.Client.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   w,
		UpdateName:   "CloseBillUpdate",
		WaitForStage: client.WorkflowUpdateStageCompleted,
		Args:         []any{closeReq.RequestID},
	})
	if err != nil {
		return &domain.Bill{}, fmt.Errorf("Error updating %s task: %v", "CloseBillUpdate", err)
	}

	var closedBill *domain.Bill
	err = updateHandle.Get(ctx, &closedBill)
	if err != nil {
		return &domain.Bill{}, fmt.Errorf("Error getting update result: %v", err)
	}

	if err := tc.CloseWorkflowSignal(ctx, w, closeReq); err != nil {
		return &domain.Bill{}, fmt.Errorf("Error closing workflow: %v", err)
	}

	return closedBill, nil
}

func (tc *TemporalClient) CloseWorkflowSignal(ctx context.Context, w string, closeReq *workflows.CloseBillSignal) error {
	err := tc.Client.SignalWorkflow(ctx, w, "", workflows.CloseWorkflowRoute.Name, closeReq)
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflows.CloseBillRoute.Name, err)
	}

	return nil
}
