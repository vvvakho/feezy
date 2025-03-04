package billing

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vvvakho/feezy/billing/service/domain"
)

func TestValidateCreateBillRequest(t *testing.T) {
	tests := []struct {
		name      string
		req       CreateBillRequest
		expectErr bool
	}{
		{"Valid Request", CreateBillRequest{UserID: uuid.NewString(), Currency: "USD"}, false},
		{"Invalid UserID", CreateBillRequest{UserID: "invalid-uuid", Currency: "USD"}, true},
		{"Invalid Currency", CreateBillRequest{UserID: uuid.NewString(), Currency: "EUR"}, true},
		{"Empty UserID", CreateBillRequest{UserID: "", Currency: "USD"}, true},
		{"Empty Currency", CreateBillRequest{UserID: uuid.NewString(), Currency: ""}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCreateBillRequest(&tc.req)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAddLineItemRequest(t *testing.T) {
	tests := []struct {
		name      string
		req       AddLineItemRequest
		expectErr bool
	}{
		{"Valid Request", AddLineItemRequest{ID: uuid.NewString(), Quantity: 2, PricePerUnit: domain.Money{Amount: 100, Currency: "USD"}}, false},
		{"Invalid ID", AddLineItemRequest{ID: "invalid-uuid", Quantity: 2, PricePerUnit: domain.Money{Amount: 100, Currency: "USD"}}, true},
		{"Invalid Quantity", AddLineItemRequest{ID: uuid.NewString(), Quantity: 0, PricePerUnit: domain.Money{Amount: 100, Currency: "USD"}}, true},
		{"Negative Quantity", AddLineItemRequest{ID: uuid.NewString(), Quantity: -1, PricePerUnit: domain.Money{Amount: 100, Currency: "USD"}}, true},
		{"Invalid Currency", AddLineItemRequest{ID: uuid.NewString(), Quantity: 2, PricePerUnit: domain.Money{Amount: 100, Currency: "EUR"}}, true},
		{"Negative Price", AddLineItemRequest{ID: uuid.NewString(), Quantity: 2, PricePerUnit: domain.Money{Amount: -100, Currency: "USD"}}, true},
		{"Empty ID", AddLineItemRequest{ID: "", Quantity: 2, PricePerUnit: domain.Money{Amount: 100, Currency: "USD"}}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAddLineItemRequest(&tc.req)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRemoveLineItemRequest(t *testing.T) {
	tests := []struct {
		name      string
		req       RemoveLineItemRequest
		expectErr bool
	}{
		{
			"Valid Request",
			RemoveLineItemRequest{
				ID:           uuid.NewString(),
				PricePerUnit: domain.Money{Amount: 100, Currency: "USD"}, // Valid Price
				Quantity:     1,                                          // Valid Quantity
			},
			false,
		},
		{
			"Invalid ID",
			RemoveLineItemRequest{
				ID:           "invalid-uuid",
				PricePerUnit: domain.Money{Amount: 100, Currency: "USD"},
				Quantity:     1,
			},
			true,
		},
		{
			"Invalid Quantity",
			RemoveLineItemRequest{
				ID:           uuid.NewString(),
				PricePerUnit: domain.Money{Amount: 100, Currency: "USD"},
				Quantity:     0,
			},
			true,
		},
		{
			"Negative Price",
			RemoveLineItemRequest{
				ID:           uuid.NewString(),
				PricePerUnit: domain.Money{Amount: -50, Currency: "USD"},
				Quantity:     1,
			},
			true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRemoveLineItemRequest(&tc.req)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCloseBillRequest(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		req       CloseBillRequest
		expectErr bool
	}{
		{"Valid Request", uuid.NewString(), CloseBillRequest{RequestID: uuid.NewString()}, false},
		{"Invalid ID", "invalid-uuid", CloseBillRequest{RequestID: uuid.NewString()}, true},
		{"Empty RequestID", uuid.NewString(), CloseBillRequest{RequestID: ""}, false},
		{"Empty ID", "", CloseBillRequest{RequestID: uuid.NewString()}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCloseBillRequest(tc.id, &tc.req)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, tc.req.RequestID)
			}
		})
	}
}
