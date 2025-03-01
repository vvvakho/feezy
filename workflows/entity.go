package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Bill struct {
	ID        uuid.UUID
	Items     []Item
	Total     Money
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

type minorUnit int64

type Money struct {
	Amount   minorUnit
	Currency string
}

type Status string

var Open Status = "BillOpen"
var Closed Status = "BillClosed"

var ValidCurrency = map[string]struct{}{
	"USD": {},
	"GEL": {},
}

var ExchangeRates = map[string]float64{
	"USD": 100,
	"GEL": 275, // 275 tetri per 100 cents
}

func IsValidCurrency(c string) (bool, error) {
	_, ok := ValidCurrency[c]
	if !ok {
		return false, errors.New("Invalid currency: " + c)
	}
	return true, nil
}

func convert(toCurrency string, fromCurrency string, amount minorUnit) (minorUnit, error) {
	// (usd, gel, 275)
	if toCurrency == fromCurrency {
		return amount, nil
	}

	// Check if exchange rates exist
	fromRate, fromExists := ExchangeRates[fromCurrency] // 2.75
	toRate, toExists := ExchangeRates[toCurrency]       // 1.0
	if !fromExists || !toExists {
		return 0, fmt.Errorf("exchange rate not defined for %s or %s", fromCurrency, toCurrency)
	}

	// Convert amount to target currency
	convertedAmount := (amount * minorUnit(toRate)) / minorUnit(fromRate) // (275 / 2.75) * 1
	return convertedAmount, nil
}
