package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vvvakho/feezy/domain"
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
		{"Invalid Currency", AddLineItemRequest{ID: uuid.NewString(), Quantity: 2, PricePerUnit: domain.Money{Amount: 100, Currency: "EUR"}}, true},
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
		{"Valid Request", RemoveLineItemRequest{ID: uuid.NewString()}, false},
		{"Invalid ID", RemoveLineItemRequest{ID: "invalid-uuid"}, true},
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
