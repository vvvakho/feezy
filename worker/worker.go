package main

import (
	"log"

	billing "github.com/vvvakho/feezy/workflows"
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
	dbConn := "postgres"

	// Instantiate Activities struct
	activities := &billing.Activities{DB: dbConn}

	// Create a worker pool for delegating tasks

	// Create a new worker listening on the create-bill-queue
	w := worker.New(c, "create-bill-queue", worker.Options{}) //TODO: save the queue names in separate file

	// Register Workflow and Activities
	w.RegisterWorkflow(billing.BillWorkflow)
	w.RegisterActivity(activities)

	// Start worker
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("Unable to start Worker: %v", err)
	}
}
