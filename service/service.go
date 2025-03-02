package service

import (
	"context"
	"fmt"

	"encore.dev/storage/sqldb"
	"github.com/vvvakho/feezy/domain"
	"github.com/vvvakho/feezy/workflow"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

//TODO: add logger

//encore:service
type Service struct {
	TemporalClient client.Client
	DB             *sqldb.Database
}

var BillsDB = sqldb.NewDatabase("bills", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

func initService() (*Service, error) {
	// Initialize Temporal Client
	c, err := initTemporalClient()
	if err != nil {
		return nil, err
	}

	return &Service{
		TemporalClient: c,
		DB:             BillsDB,
	}, nil
}

func initTemporalClient() (client.Client, error) {
	// Connect to Temporal
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to Temporal: %v", err)
	}
	return c, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.TemporalClient.Close()
}

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

func AddLineItemSignal(ctx context.Context, c client.Client, w string, billItem *domain.Item) error {
	err := c.SignalWorkflow(ctx, w, "", workflow.AddLineItemRoute.Name, workflow.AddItemSignal{LineItem: *billItem})
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflow.AddLineItemRoute.Name, err)
	}

	return nil
}

func RemoveLineItemSignal(ctx context.Context, c client.Client, w string, billItem *domain.Item) error {
	err := c.SignalWorkflow(ctx, w, "", workflow.RemoveLineItemRoute.Name, workflow.AddItemSignal{LineItem: *billItem})
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflow.RemoveLineItemRoute.Name, err)
	}

	return nil
}

func CloseBillSignal(ctx context.Context, c client.Client, w string, payload *workflow.CloseBillSignal) error {
	err := c.SignalWorkflow(ctx, w, "", workflow.CloseBillRoute.Name, payload)
	if err != nil {
		return fmt.Errorf("Error signaling %s task: %v", workflow.CloseBillRoute.Name, err)
	}

	return nil
}
