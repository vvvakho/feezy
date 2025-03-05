package conf

import "go.temporal.io/sdk/client"

var WORKER_DB_CONN = "postgresql://feezy-zyei:local@127.0.0.1:9500/bills?sslmode=disable"

var TEMPORAL_CLIENT_CONF = client.Options{
	// Add connection options here
}
