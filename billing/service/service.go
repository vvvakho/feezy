package billing

import (
	"context"
	"fmt"
	"time"

	"encore.dev/storage/cache"
	_ "github.com/alicebob/miniredis/v2"
	"github.com/vvvakho/feezy/billing/service/domain"
	"github.com/vvvakho/feezy/billing/service/execution"
	"github.com/vvvakho/feezy/billing/workflows"
)

//encore:service
type Service struct {
	Execution  Execution
	Repository Repository
	Cache      *cache.StructKeyspace[string, GetBillResponse]
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

var c = cache.NewCluster("billing-cache-cluster", cache.ClusterConfig{
	EvictionPolicy: cache.AllKeysLRU, // Least recently used keys eviction policy
})

// Initialize billing service with an Execution and Repository entities
func initService() (*Service, error) {

	billCache := cache.NewStructKeyspace[string, GetBillResponse](c, cache.KeyspaceConfig{
		KeyPattern:    "bill_cache/:key",
		DefaultExpiry: cache.ExpireIn(10 * time.Minute),
	})

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
		Cache:      billCache,
	}, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.Execution.Close()
}
