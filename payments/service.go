package payments

import (
	"context"
	"fmt"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/rlog"
	billing "github.com/vvvakho/feezy/billing/service"
	"github.com/vvvakho/feezy/billing/service/domain"
	bDomain "github.com/vvvakho/feezy/billing/service/domain"
)

// PaymentService handles payment-related operations
//
//encore:service
type Service struct {
}

// Initialize the payment service
func initService() (*Service, error) {
	return &Service{}, nil
}

// PayBillRequest represents the request to pay a bill.
type PayBillRequest struct {
	BillID   string `json:"bill_id"`
	UserID   string `json:"user_id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// PayBillResponse represents the response from a bill payment.
type PayBillResponse struct {
	Bill    *domain.Bill
	Message string `json:"message"`
}

// PayBill processes a payment for a given bill.
//
//encore:api private method=POST path=/payments/pay
func (s *Service) PayBill(ctx context.Context, req *PayBillRequest) (*PayBillResponse, error) {

	// Fetch the bill details from the Billing Service
	bill, err := billing.GetBill(ctx, req.BillID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving bill: %v", err)
	}

	// Check if the bill is open
	if bill.Status != bDomain.BillOpen {
		return nil, &errs.Error{
			Code:    errs.FailedPrecondition,
			Message: "cannot pay a closed or non-existent bill",
		}
	}

	// Simulate payment processing (replace with real payment gateway integration)
	rlog.Info("Processing payment for bill", "BillID", req.BillID, "Amount", req.Amount, "Currency", req.Currency)
	time.Sleep(2 * time.Second) // Simulating external API call

	// Call the Billing Service to close the bill after payment
	resp, err := billing.CloseBill(ctx, req.BillID, &billing.CloseBillRequest{
		RequestID: req.BillID, // Use bill ID as request ID for idempotency
	})
	if err != nil {
		return nil, fmt.Errorf("error closing bill after payment: %v", err)
	}

	return &PayBillResponse{Bill: resp.Bill, Message: resp.Status}, nil
}
