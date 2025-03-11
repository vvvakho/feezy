package main

import (
	"database/sql"
	"log"

	_ "github.com/alicebob/miniredis/v2"
	_ "github.com/lib/pq"
	"github.com/vvvakho/feezy/billing/conf"
	"github.com/vvvakho/feezy/billing/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Initialize a Temporal worker to handle bill creation and
// connect to PostgreSQL independently for horizontal scalability.
// Register and listen for tasks associated with "create-bill-queue".
func main() {
	// Connect to Temporal
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}

	// Create a worker pool for delegating tasks
	w := worker.New(c, "create-bill-queue", worker.Options{})

	// Establishing a separate DB connection lets us decouple from Encore
	// in case we want to introduce independent horizontal worker scaling
	postgres, err := sql.Open("postgres", conf.WORKER_DB_CONN)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer postgres.Close()

	db := workflows.Repo{
		DB: postgres,
	}

	activities := &workflows.Activities{
		Repository: &db,
	}

	// Register Workflow and Activities
	w.RegisterWorkflow(workflows.BillWorkflow)
	w.RegisterActivity(activities)

	// Start worker
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("Unable to start Worker: %v", err)
	}
}

// Dependency injection -- primarily for mock testing
var TemporalDial = client.Dial
