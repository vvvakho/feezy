package billing

import (
	"context"
	"fmt"

	"encore.dev/storage/sqldb"
	tc "github.com/vvvakho/feezy/billing/service/temporal"
	"go.temporal.io/sdk/client"
)

//encore:service
type Service struct {
	TemporalClient client.Client
	Repository     Repository
}

var BillsDB = sqldb.NewDatabase("bills", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

func initService() (*Service, error) {
	tc, err := tc.InitTemporalClient()
	if err != nil {
		return &Service{}, fmt.Errorf("Unable to connect to Temporal: %v", err)
	}

	db := &Repo{DB: BillsDB}

	return &Service{
		TemporalClient: tc,
		Repository:     db,
	}, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.TemporalClient.Close()
}
