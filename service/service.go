package service

import (
	"context"
	"fmt"

	"encore.dev/storage/sqldb"
	"github.com/vvvakho/feezy/activity"
	"github.com/vvvakho/feezy/workflow"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

//TODO: add logger

//encore:service
type Service struct {
	TemporalClient *client.Client
	Worker         *worker.Worker
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

	// Initialize Temporal worker
	w, err := initWorker(c)
	if err != nil {
		return nil, err
	}

	return &Service{
		TemporalClient: &c,
		Worker:         w,
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

func initWorker(c client.Client) (*worker.Worker, error) {
	// Create a worker pool for delegating tasks
	w := worker.New(c, "create-bill-queue", worker.Options{})

	// Instantiate Activities struct
	activities := &activity.Activities{
		DB: BillsDB,
	}

	// Register Workflow and Activities
	w.RegisterWorkflow(workflow.BillWorkflow)
	w.RegisterActivity(activities)

	// Start worker asynchronously
	go func() {
		if err := w.Run(worker.InterruptCh()); err != nil {
			fmt.Printf("Error starting Temporal Worker: %v\n", err)
		}
	}()

	return &w, nil
}

func (s *Service) Shutdown(force context.Context) {
	temporal := *s.TemporalClient
	temporal.Close()

	worker := *s.Worker
	worker.Stop()
}

func isWorkflowRunning(c client.Client, workflowID string) (bool, error) {
	response, err := c.DescribeWorkflowExecution(context.Background(), workflowID, "")
	if err != nil {
		return false, err
	}
	return response.WorkflowExecutionInfo.Status == enums.WORKFLOW_EXECUTION_STATUS_RUNNING, nil
}
