package billing

import (
	"time"

	"github.com/google/uuid"
)

type Bill struct {
	ID        uuid.UUID
	Items     []Item
	Total     Money
	Currency  string
	Status    Status
	UserID    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Item struct {
	ID           uuid.UUID
	Quantity     int64
	Description  string
	PricePerUnit Money
}

type AddItemSignal struct {
	LineItem Item
}

type RemoveItemSignal struct {
	LineItem Item
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
