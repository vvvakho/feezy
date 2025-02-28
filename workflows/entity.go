package billing

import (
	"time"

	"github.com/google/uuid"
)

type Bill struct {
	ID        uuid.UUID
	Items     []LineItem
	Total     Money
	Currency  string
	Status    Status
	UserID    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LineItem struct {
	ID           uuid.UUID
	Quantity     int64
	Description  string
	PricePerUnit Money
}

type AddLineItemSignal struct {
	Route    string
	LineItem LineItem
}

type RemoveLineItemSignal struct {
	Route    string
	LineItem LineItem
}

type CloseBillSignal struct {
	Route string
}

type Money struct {
	Amount   int64
	Currency string
}

type Status string

var Open Status = "BillOpen"
var Closed Status = "BillClosed"
