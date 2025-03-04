# Feezy - Fees API for Managing Bills

## Overview
Feezy is a **billing and fees management API** that provides an efficient way to handle **bills, line items, and transactions**. It integrates **Temporal Workflows** to ensure reliability and fault tolerance, enabling smooth and scalable billing operations.

## Features
- **Bill Creation**: Users can create bills in multiple currencies.
- **Line Item Management**: Add or remove line items dynamically.
- **Bill Retrieval**: Fetch open or closed bills from the database.
- **Bill Closure**: Finalize a bill, preventing further modifications.
- **Temporal Workflow Integration**: Handles asynchronous operations reliably.
- **PostgreSQL Database**: Efficiently stores open and closed bills.

## Tech Stack
- **Language**: Golang
- **Database**: PostgreSQL
- **Workflow Orchestration**: Temporal
- **Framework**: Encore (for API development)
- **Storage**: `encore.dev/storage/sqldb`

## Project Structure
```
feezy/
├── billing/
│   ├── conf/
│   ├── service/
│   │   ├── domain/
│   │   │   └── domain.go    # Core domain models (Bill, Item, Money)
│   │   ├── migrations/
│   │   ├── temporal/
│   │   │   └── client.go    # Temporal client for workflow interactions
│   │   ├── api.go           # API endpoints for bill management
│   │   ├── db.go            # Database repository for bill storage
│   │   ├── dto.go           # Payload parameters for api requests
│   │   └── service.go       # Business logic and service layer
│   ├── worker/
│   │   └── worker.go        # Temporal worker setup
│   ├── workflows/
│   │   ├── activity.go      # Activity functions for database operations
│   │   ├── signals.go       # Workflow signal handlers
│   │   ├── workflow.go      # Temporal workflow definition
├── payments/                # Placeholder paymnets service
├── notifications/           # Placeholder notificaitons service
```

## API Endpoints
### 1. Create a Bill
```
POST /bills
```
**Request:**
```json
{
  "user_id": "<UUID>",
  "currency": "USD"
}
```
- `user_id`: The unique identifier of the user creating the bill.
- `currency`: The currency code for the bill (e.g., USD, GEL). Must be a valid currency.

**Response:**
```json
{
  "id": "<UUID>",
  "user_id": "<UUID>",
  "currency": "USD",
  "created_at": "<timestamp>",
  "status": "BillOpen"
}
```
- `id`: Unique bill identifier.
- `user_id`: ID of the user associated with the bill.
- `currency`: The currency code.
- `created_at`: Timestamp of bill creation.
- `status`: The status of the bill (`BillOpen`, `BillClosed`).

### 2. Get a Bill
```
GET /bills/:id
```
**Response:**
```json
{
  "id": "<UUID>",
  "items": [],
  "total": { "amount": 0, "currency": "USD" },
  "status": "BillOpen",
  "user_id": "<UUID>",
  "created_at": "<timestamp>",
  "updated_at": "<timestamp>"
}
```
- `id`: Unique bill identifier.
- `items`: List of associated bill items.
- `total`: Total cost in the bill’s currency.
- `status`: Bill status (`BillOpen`, `BillClosed`).
- `user_id`: ID of the user associated with the bill.
- `created_at`: Timestamp of bill creation.
- `updated_at`: Timestamp of last update.

### 3. Add Line Item
```
POST /bills/:id/items
```
**Request:**
```json
{
  "id": "<UUID>",
  "quantity": 2,
  "description": "Service Fee",
  "price_per_unit": { "amount": 100, "currency": "USD" }
}
```
- `id`: The bill ID to which the item should be added.
- `quantity`: The number of units for this line item (must be 1 or greater).
- `description`: A short description of the line item.
- `price_per_unit.amount`: The price per unit in minor currency units.
- `price_per_unit.currency`: The currency code (must match the bill’s currency).

**Response:**
```json
{
  "message": "Request has been sent"
}
```

### 4. Remove Line Item
```
PATCH /bills/:id/items
```

### 5. Close Bill
```
PATCH /bills/:id
```

## Why Temporal Workflows?
Temporal Workflows are a **crucial component** of Feezy’s architecture due to their ability to **persistently manage long-running operations**. The nature of billing requires **stateful tracking** of bills, which is best handled by a workflow engine rather than a traditional stateless request-response cycle. Key benefits include:

- **Reliability**: Temporal ensures all steps in a bill’s lifecycle (creation, modification, closure) are executed successfully, with automatic retries on failures.
- **Consistency in Distributed Systems**: Feezy operates in a **microservice-based architecture**, where failures, network latencies, or restarts can disrupt billing processes. Temporal guarantees that billing tasks continue executing even after system crashes.
- **Stateful Representation**: Each bill is maintained within a Temporal workflow, meaning all updates (adding/removing line items) are processed in a controlled, logical sequence, avoiding race conditions or conflicts.
- **Idempotency**: Ensures that duplicate requests (e.g., retrying the same close bill request) do not lead to inconsistent billing states.
- **Asynchronous Processing**: Adding or removing items doesn’t block API requests, as these actions are queued within Temporal and processed reliably.

## Setup & Running
### Prerequisites
- Go 1.20+
- PostgreSQL
- Temporal Server
- Encore CLI

### Install Feezy
```sh
git clone https://github.com/vvvakho/feezy.git
cd feezy
go mod tidy
```

### Running Temporal Server
```sh
temporal server start-dev
```

### Running the API
```sh
encore run
```

### Running Temporal Worker
```sh
go run worker.go
```
