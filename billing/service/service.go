package billing

import (
	"context"
	"fmt"

	"github.com/vvvakho/feezy/billing/service/domain"
	"github.com/vvvakho/feezy/billing/service/execution"
	"github.com/vvvakho/feezy/billing/workflows"
)

//encore:service
type Service struct {
	Execution  Execution
	Repository Repository
}

// Interface for the Execution entity
type Execution interface {
	CreateBillWorkflow(context.Context, *domain.Bill) error
	GetBillQuery(context.Context, string, *domain.Bill) error
	IsWorkflowRunning(string) error
	AddLineItemSignal(context.Context, string, *domain.Item) error
	RemoveLineItemSignal(context.Context, string, *domain.Item) error
	CloseBillUpdate(context.Context, string, *workflows.CloseBillSignal) (*domain.Bill, error)
	Close()
}

// Interface for the Repository entity
type Repository interface {
	GetOpenBillFromDB(context.Context, string) (*domain.Bill, error)
	GetClosedBillFromDB(context.Context, string) (*domain.Bill, error)
	GetClosedBillItemsFromDB(context.Context, string) ([]domain.Item, error)
}

// Initialize billing service with an Execution and Repository entities
func initService() (*Service, error) {
	// Init Execution client
	tc, err := execution.New()
	if err != nil {
		return &Service{}, fmt.Errorf("Unable to initialize Temporal: %v", err)
	}

	// Init Repository
	db, err := NewRepo()
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize Repository: %v", err)
	}

	// Init Service
	return &Service{
		Execution:  tc,
		Repository: db,
	}, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.Execution.Close()
}
