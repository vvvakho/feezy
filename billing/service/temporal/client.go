package temporalclient

import (
	"context"
	"fmt"

	"github.com/vvvakho/feezy/billing/service/domain"
	"github.com/vvvakho/feezy/billing/workflows"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// Dependency injection -- primarily for mock testing
var TemporalDial = client.Dial

func InitTemporalClient() (client.Client, error) {
	// Connect to Temporal
	c, err := TemporalDial(client.Options{})
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to Temporal: %v", err)
	}
	return c, nil
}

func IsWorkflowRunning(c client.Client, workflowsID string) error {
	response, err := c.DescribeWorkflowExecution(context.Background(), workflowsID, "")
	if err != nil {
		return err
	}
	if response.WorkflowExecutionInfo.Status != enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return fmt.Errorf("Workflow not running")
	}
	return nil
}

func GetBillQuery(ctx context.Context, c client.Client, w string, bill *domain.Bill) error {
	// Start signal synchronously
	resp, err := c.QueryWorkflow(ctx, w, "", "getBill")
	if err != nil {
		return fmt.Errorf("Unable to initiate query signal: %v", err)
	}
	err = resp.Get(&bill)
	if err != nil {
		return fmt.Errorf("Unable to parse query response into Bill: %v", err)
	}

	return nil
}

func CreateBillWorkflow(ctx context.Context, c client.Client, bill *domain.Bill) error {
	_, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        bill.ID.String(), //TODO: may need to edit workflows id to not just be bill id
		TaskQueue: "create-bill-queue",
	}, workflows.BillWorkflow, bill)

	if err != nil {
		return fmt.Errorf("Unable to initiate workflows: %v", err)
	}

	return nil
}

func AddLineItemSignal(ctx context.Context, c client.Client, w string, billItem *domain.Item) error {
	err := c.SignalWorkflow(ctx, w, "", workflows.AddLineItemRoute.Name, workflows.AddItemSignal{LineItem: *billItem})
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflows.AddLineItemRoute.Name, err)
	}

	return nil
}

func RemoveLineItemSignal(ctx context.Context, c client.Client, w string, billItem *domain.Item) error {
	err := c.SignalWorkflow(ctx, w, "", workflows.RemoveLineItemRoute.Name, workflows.AddItemSignal{LineItem: *billItem})
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflows.RemoveLineItemRoute.Name, err)
	}

	return nil
}

func CloseBillSignal(ctx context.Context, c client.Client, w string, closeReq *workflows.CloseBillSignal) error {
	err := c.SignalWorkflow(ctx, w, "", workflows.CloseBillRoute.Name, closeReq)
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflows.CloseBillRoute.Name, err)
	}

	return nil
}

func CloseBillUpdate(ctx context.Context, c client.Client, w string, closeReq *workflows.CloseBillSignal) (*domain.Bill, error) {
	updateHandle, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
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

	if err := CloseWorkflowSignal(ctx, c, w, closeReq); err != nil {
		return &domain.Bill{}, fmt.Errorf("Error closing workflow: %v", err)
	}

	return closedBill, nil
}

func CloseWorkflowSignal(ctx context.Context, c client.Client, w string, closeReq *workflows.CloseBillSignal) error {
	err := c.SignalWorkflow(ctx, w, "", workflows.CloseWorkflowRoute.Name, closeReq)
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflows.CloseBillRoute.Name, err)
	}

	return nil
}
