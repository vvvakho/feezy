package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewBill(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		currency  string
		expectErr bool
	}{
		{"Valid User and Currency", uuid.New().String(), "USD", false},
		{"Invalid User ID", "invalid-uuid", "USD", true},
		{"Invalid Currency", uuid.New().String(), "EUR", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bill, err := NewBill(tc.userID, tc.currency)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, bill.Total.Currency, tc.currency)
			}
		})
	}
}

func TestIsValidCurrency(t *testing.T) {
	tests := []struct {
		currency  string
		expectErr bool
	}{
		{"USD", false},
		{"GEL", false},
		{"EUR", true},
	}

	for _, tc := range tests {
		t.Run(tc.currency, func(t *testing.T) {
			valid, err := IsValidCurrency(tc.currency)
			if tc.expectErr {
				assert.Error(t, err)
				assert.False(t, valid)
			} else {
				assert.NoError(t, err)
				assert.True(t, valid)
			}
		})
	}
}

func TestConvert(t *testing.T) {
	tests := []struct {
		from      string
		to        string
		amount    MinorUnit
		expectErr bool
		expected  MinorUnit
	}{
		{"USD", "GEL", 100, false, 275},
		{"GEL", "USD", 275, false, 100},
		{"USD", "USD", 100, false, 100},
		{"USD", "EUR", 100, true, 0},
	}

	for _, tc := range tests {
		t.Run(tc.from+" to "+tc.to, func(t *testing.T) {
			converted, err := Convert(tc.to, tc.from, tc.amount)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, converted, tc.expected)
			}
		})
	}
}

func TestAddLineItem(t *testing.T) {
	bill, _ := NewBill(uuid.New().String(), "USD")
	tests := []struct {
		name        string
		item        Item
		expectTotal MinorUnit
		expectErr   bool
	}{
		{"Add New Item", Item{ID: uuid.New(), Quantity: 2, Description: "Item 1", PricePerUnit: Money{Amount: 50, Currency: "USD"}}, 100, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := bill.AddLineItem(tc.item)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, bill.Total.Amount, tc.expectTotal)
			}
		})
	}
}

func TestRemoveLineItem(t *testing.T) {
	bill, _ := NewBill(uuid.New().String(), "USD")
	item := Item{
		ID: uuid.New(), Quantity: 2, Description: "Item 1", PricePerUnit: Money{Amount: 50, Currency: "USD"},
	}
	bill.AddLineItem(item)
	tests := []struct {
		name        string
		removeItem  Item
		expectTotal MinorUnit
		expectErr   bool
	}{
		{"Remove One Quantity", Item{ID: item.ID, Quantity: 1, PricePerUnit: item.PricePerUnit}, 50, false},
		{"Remove Remaining Quantity", Item{ID: item.ID, Quantity: 1, PricePerUnit: item.PricePerUnit}, 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := bill.RemoveLineItem(tc.removeItem)
			assert.NoError(t, err)
			assert.Equal(t, bill.Total.Amount, tc.expectTotal)
		})
	}
}

func TestBillTimestamps(t *testing.T) {
	bill, _ := NewBill(uuid.New().String(), "USD")
	tests := []struct {
		name      string
		check     func() bool
		expectErr bool
	}{
		{"Check CreatedAt", func() bool { return time.Since(bill.CreatedAt) < time.Second }, false},
		{"Check UpdatedAt", func() bool { return time.Since(bill.UpdatedAt) < time.Second }, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, tc.check())
		})
	}
}
