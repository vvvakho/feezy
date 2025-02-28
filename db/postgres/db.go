package db

import "encore.dev/storage/sqldb"

// TODO: decouple with an interface
var billsDB = sqldb.NewDatabase("bills", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})
