package worker

import (
	"context"
	"fmt"

	"github.com/vvvakho/feezy/activity"
	"github.com/vvvakho/feezy/workflow"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

//encore:service
type Service struct {
	temporalClient client.Client
	worker         worker.Worker
}

// Initialize the Temporal worker inside Encore
func initService() (*Service, error) {
	// Connect to Temporal
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil, err
	}

	// Create a worker pool for delegating tasks
	w := worker.New(c, "create-bill-queue", worker.Options{})

	// Instantiate Activities struct
	activities := &activity.Activities{}

	// Register Workflow and Activities
	w.RegisterWorkflow(workflow.BillWorkflow)
	w.RegisterActivity(activities)

	// Start worker asynchronously (so Encore doesn't block)
	err = w.Start()
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("start temporal worker: %v", err)
	}

	return &Service{temporalClient: c, worker: w}, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.temporalClient.Close()
	s.worker.Stop()
}
