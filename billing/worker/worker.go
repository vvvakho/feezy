package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"github.com/vvvakho/feezy/billing/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Connect to Temporal
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}

	// Establishing a separate DB connection lets us decouple from Encore
	// in case we want to introduce independent horizontal worker scaling
	postgres, err := sql.Open("postgres", "postgresql://feezy-zyei:local@127.0.0.1:9500/bills?sslmode=disable") //TODO: conn string from conf
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

	// Worker for handling bill creation
	createBillWorker := worker.New(c, "create-bill-queue", worker.Options{})
	createBillWorker.RegisterWorkflow(workflows.BillWorkflow)
	createBillWorker.RegisterActivity(activities.AddOpenBillToDB)

	// Worker for handling line item modifications (add/remove)
	modifyBillWorker := worker.New(c, "modify-bill-queue", worker.Options{})
	modifyBillWorker.RegisterWorkflow(workflows.BillWorkflow)
	modifyBillWorker.RegisterActivity(activities.AddOpenBillToDB) // Ensure bill updates reflect in DB

	// Worker for handling bill closure
	closeBillWorker := worker.New(c, "close-bill-queue", worker.Options{})
	closeBillWorker.RegisterWorkflow(workflows.BillWorkflow)
	closeBillWorker.RegisterActivity(activities.AddClosedBillToDB)

	// Start all workers concurrently
	go runWorker(createBillWorker, "create-bill-queue")
	go runWorker(modifyBillWorker, "modify-bill-queue")
	go runWorker(closeBillWorker, "close-bill-queue")

	// Keep the process running
	select {}
}

// runWorker starts a Temporal worker and logs if it fails
func runWorker(w worker.Worker, queueName string) {
	log.Printf("Starting worker for %s...", queueName)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("Unable to start worker for %s: %v", queueName, err)
	}
}
