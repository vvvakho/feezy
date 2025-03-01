package main

import (
	"log"

	db "github.com/vvvakho/feezy/db/postgres"
	"github.com/vvvakho/feezy/workflow"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Connect to Temporal
	c, err := client.Dial(client.Options{}) //TODO: save the client options in config
	if err != nil {
		log.Fatalf("Unable to connect to Temporal server: %v", err)
	}
	defer c.Close()

	// Initialize DB connection
	//TODO: DB using Encore

	// Instantiate Activities struct
	activities := &workflow.Activities{DB: &db.PostgresBillStorage{}}

	// Create a worker pool for delegating tasks

	// Create a new worker listening on the create-bill-queue
	w := worker.New(c, "create-bill-queue", worker.Options{}) //TODO: save the queue names in separate file

	// Register Workflow and Activities
	w.RegisterWorkflow(workflow.BillWorkflow)
	w.RegisterActivity(activities)

	// Start worker
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("Unable to start Worker: %v", err)
	}
}
