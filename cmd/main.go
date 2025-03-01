package main

import (
	"context"

	"github.com/vvvakho/feezy/activity"
	"github.com/vvvakho/feezy/api"
	"github.com/vvvakho/feezy/workflow"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

//encore:service
type BillingService struct {
	temporalClient *client.Client
	worker         *worker.Worker
	server         *api.Server
}

func initService() (*BillingService, error) {
	// Initialize Temporal Client
	c := initTemporalClient()

	// Initialize API Server
	s := initServer()

	// Initialize Temporal worker
	w := initWorker(*c, *s)

	return &BillingService{
		temporalClient: c,
		worker:         w,
		server:         s,
	}, nil
}

func initTemporalClient() *client.Client {
	// Connect to Temporal
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil //TODO: fatal
	}
	return &c
}

func initWorker(c client.Client, s api.Server) *worker.Worker {
	// Create a worker pool for delegating tasks
	w := worker.New(c, "create-bill-queue", worker.Options{})

	// Instantiate Activities struct
	activities := &activity.Activities{
		Server: &s,
	}

	// Register Workflow and Activities
	w.RegisterWorkflow(workflow.BillWorkflow)
	w.RegisterActivity(activities)

	// Start worker asynchronously (so Encore doesn't block)
	err := w.Start()
	if err != nil {
		c.Close()
		return nil //TODO: panic
	}

	return &w
}

func initServer() *api.Server {
	return &api.Server{
		DB: api.InitDB(),
	}
}

func (s *BillingService) Shutdown(force context.Context) {
	temporal := *s.temporalClient
	temporal.Close()

	worker := *s.worker
	worker.Stop()
}
