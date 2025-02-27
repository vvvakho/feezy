package fees

import (
	"time"

	"github.com/google/uuid"
)

type BillState struct {
	ID        uuid.UUID
	Items     []LineItem
	Total     Money
	Status    BillStatus
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

type Money struct {
	Amount   int64
	Currency string
}

type BillStatus string

var BillOpen BillStatus = "BillOpen"
var BillClosed BillStatus = "BillClosed"
