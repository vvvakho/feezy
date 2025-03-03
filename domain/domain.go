package domain

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

type Bill struct {
	ID        uuid.UUID
	Items     []Item
	Total     Money
	Status    Status
	UserID    uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	ClosedAt  time.Time
}

type Item struct {
	ID           uuid.UUID
	Quantity     int64
	Description  string
	PricePerUnit Money
}

type MinorUnit int64

type Money struct {
	Amount   MinorUnit
	Currency string
}

type Status string

var BillOpen Status = "BillOpen"
var BillClosing Status = "BillClosing"
var BillClosed Status = "BillClosed"

var ValidCurrency = map[string]struct{}{
	"USD": {},
	"GEL": {},
}

var ExchangeRates = map[string]float64{
	"USD": 100,
	"GEL": 275, // 275 tetri per 100 cents
}

func NewBill(userID string, currency string) (*Bill, error) {
	billID, err := uuid.NewV7()
	if err != nil {
		return &Bill{}, fmt.Errorf("Unable to initialize bill ID: %v", err)
	}

	parseID, err := uuid.Parse(userID)
	if err != nil {
		return &Bill{}, fmt.Errorf("Invalid UserID")
	}

	return &Bill{
		ID:        billID,
		UserID:    parseID,
		Items:     []Item{},
		Total:     Money{Amount: 0, Currency: currency},
		Status:    BillOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func IsValidCurrency(c string) (bool, error) {
	_, ok := ValidCurrency[c]
	if !ok {
		return false, errors.New("Invalid currency: " + c)
	}
	return true, nil
}

func Convert(toCurrency string, fromCurrency string, amount MinorUnit) (MinorUnit, error) {
	if toCurrency == fromCurrency {
		return amount, nil
	}

	// Check if exchange rates exist
	fromRate, fromExists := ExchangeRates[fromCurrency]
	toRate, toExists := ExchangeRates[toCurrency]
	if !fromExists || !toExists {
		return 0, fmt.Errorf("exchange rate not defined for %s or %s", fromCurrency, toCurrency)
	}

	// Convert amount to target currency
	convertedAmount := (amount * MinorUnit(toRate)) / MinorUnit(fromRate)
	return convertedAmount, nil
}

func (b *Bill) AddLineItem(itemToAdd Item) error {
	for i, itemInBill := range b.Items {
		if itemInBill.ID == itemToAdd.ID {
			if itemInBill.PricePerUnit != itemToAdd.PricePerUnit {
				return errors.New("Price of item has changed, please use new UUID")
			}

			b.Items[i].Quantity += itemToAdd.Quantity
			if err := b.CalculateTotal(); err != nil {
				return err
			}

			return nil
		}
	}
	b.Items = append(b.Items, itemToAdd)

	if err := b.CalculateTotal(); err != nil {
		return err
	}

	return nil
}

func (b *Bill) RemoveLineItem(itemToRemove Item) error {
	for i, itemInBill := range b.Items {
		if itemInBill.ID == itemToRemove.ID {
			if itemInBill.PricePerUnit != itemToRemove.PricePerUnit {
				return errors.New("Price of item has changed, please use new UUID")
			}

			b.Items[i].Quantity -= itemToRemove.Quantity
			if b.Items[i].Quantity <= 0 {
				b.Items = slices.Delete(b.Items, i, i+1)
			}
		}
	}

	if err := b.CalculateTotal(); err != nil {
		return err
	}

	return nil
}

func (b *Bill) CalculateTotal() error {
	var total MinorUnit
	for _, v := range b.Items {
		amount := v.PricePerUnit.Amount
		fromCurrency := v.PricePerUnit.Currency
		toCurrency := b.Total.Currency

		unitPrice, err := Convert(toCurrency, fromCurrency, amount)
		if err != nil {
			return err
		}

		total += unitPrice * MinorUnit(v.Quantity)
	}
	b.Total.Amount = total

	return nil
}
