package billing

import (
	"context"
	"fmt"

	"encore.dev/storage/sqldb"
	"go.temporal.io/sdk/client"
)

//TODO: add logger

//encore:service
type Service struct {
	TemporalClient client.Client
	DBencore       *sqldb.Database
}

var BillsDB = sqldb.NewDatabase("bills", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

func initService() (*Service, error) {
	// var c client.Client
	// var err error
	// var backoff time.Duration = time.Second

	// // Retry Temporal connection with exponential backoff
	// for attempts := 0; ; attempts++ { // Max 5 attempts before failure
	// 	c, err = initTemporalClient()
	// 	if err == nil {
	// 		break
	// 	}

	// 	log.Printf("Temporal unavailable, retrying in %v: %v", backoff, err)
	// 	time.Sleep(backoff)

	// 	if backoff < 32*time.Second {
	// 		backoff *= 2 // Exponential backoff
	// 	}
	// }

	c, err := initTemporalClient()
	if err != nil {
		return &Service{}, fmt.Errorf("Unable to connect to Temporal: %v", err)
	}

	return &Service{
		TemporalClient: c,
		DBencore:       BillsDB,
	}, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.TemporalClient.Close()
}
