package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"github.com/vvvakho/feezy/workflow"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Connect to Temporal
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err) //TODO: fatal
	}
	// Create a worker pool for delegating tasks
	w := worker.New(c, "create-bill-queue", worker.Options{})

	// Manually establishing connection lets us decouple from Encore
	// in case we want to introduce independent horizontal worker scaling
	db, err := sql.Open("postgres", "postgresql://feezy-zyei:local@127.0.0.1:9500/bills?sslmode=disable") //TODO: conn string from conf
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	// Instantiate Activities struct
	activities := &workflow.Activities{
		DB: db,
	}

	// Register Workflow and Activities
	w.RegisterWorkflow(workflow.BillWorkflow)
	w.RegisterActivity(activities)

	// Start worker
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("Unable to start Worker: %v", err)
	}
}
