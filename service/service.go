package service

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
	// Initialize Temporal Client
	c, err := initTemporalClient()
	if err != nil {
		return nil, err
	}

	return &Service{
		TemporalClient: c,
		DBencore:       BillsDB,
	}, nil
}

func initTemporalClient() (client.Client, error) {
	// Connect to Temporal
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to Temporal: %v", err)
	}
	return c, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.TemporalClient.Close()
}
