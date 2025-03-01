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

type minorUnit int64

type Money struct {
	Amount   minorUnit
	Currency string
}

type Status string

var BillOpen Status = "BillOpen"
var BillClosed Status = "BillClosed"

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

func (b *Bill) AddLineItem(itemToAdd Item) error {
	for i, itemInBill := range b.Items {
		if itemInBill.ID == itemToAdd.ID {
			if itemInBill.PricePerUnit != itemToAdd.PricePerUnit {
				return errors.New("Price of item has changed, please use new UUID")
			}

			b.Items[i].Quantity += itemToAdd.Quantity
			return nil
		}
	}
	b.Items = append(b.Items, itemToAdd)

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

	return nil
}

func (b *Bill) CalculateTotal() error {
	var total minorUnit
	for _, v := range b.Items {
		amount := v.PricePerUnit.Amount         // 275 gel
		fromCurrency := v.PricePerUnit.Currency // gel
		toCurrency := b.Total.Currency          // usd

		unitPrice, err := convert(toCurrency, fromCurrency, amount) // 100
		if err != nil {
			return err
		}

		total += unitPrice * minorUnit(v.Quantity) // 100 * 1
	}
	b.Total.Amount = total

	return nil
}
