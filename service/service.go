package service

import (
	"context"
	"fmt"

	"encore.dev/storage/sqldb"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

//TODO: add logger

//encore:service
type Service struct {
	TemporalClient *client.Client
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
		TemporalClient: &c,
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
	temporal := *s.TemporalClient
	temporal.Close()
}

func isWorkflowRunning(c client.Client, workflowID string) (bool, error) {
	response, err := c.DescribeWorkflowExecution(context.Background(), workflowID, "")
	if err != nil {
		return false, err
	}
	return response.WorkflowExecutionInfo.Status == enums.WORKFLOW_EXECUTION_STATUS_RUNNING, nil
}
