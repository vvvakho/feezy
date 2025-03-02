package service

import (
	"context"
	"fmt"

	"github.com/vvvakho/feezy/domain"
	"github.com/vvvakho/feezy/workflow"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

func isWorkflowRunning(c client.Client, workflowID string) error {
	response, err := c.DescribeWorkflowExecution(context.Background(), workflowID, "")
	if err != nil {
		return err
	}
	if response.WorkflowExecutionInfo.Status != enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return fmt.Errorf("Workflow not running")
	}
	return nil
}

func getBillQuery(ctx context.Context, c client.Client, w string, bill *domain.Bill) error {
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

func createBillWorkflow(ctx context.Context, c client.Client, bill *domain.Bill) error {
	_, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        bill.ID.String(), //TODO: may need to edit workflow id to not just be bill id
		TaskQueue: "create-bill-queue",
	}, workflow.BillWorkflow, bill)

	if err != nil {
		return fmt.Errorf("Unable to initiate workflow: %v", err)
	}

	return nil
}

func addLineItemSignal(ctx context.Context, c client.Client, w string, billItem *domain.Item) error {
	err := c.SignalWorkflow(ctx, w, "", workflow.AddLineItemRoute.Name, workflow.AddItemSignal{LineItem: *billItem})
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflow.AddLineItemRoute.Name, err)
	}

	return nil
}

func removeLineItemSignal(ctx context.Context, c client.Client, w string, billItem *domain.Item) error {
	err := c.SignalWorkflow(ctx, w, "", workflow.RemoveLineItemRoute.Name, workflow.AddItemSignal{LineItem: *billItem})
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflow.RemoveLineItemRoute.Name, err)
	}

	return nil
}

func closeBillSignal(ctx context.Context, c client.Client, w string, payload *workflow.CloseBillSignal) error {
	err := c.SignalWorkflow(ctx, w, "", workflow.CloseBillRoute.Name, payload)
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflow.CloseBillRoute.Name, err)
	}

	return nil
}
