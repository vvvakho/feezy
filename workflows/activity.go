package billing

import (
	"fmt"

	"golang.org/x/net/context"
)

type Activities struct {
	DB string
}

func (a *Activities) AddToDB(ctx context.Context, bill Bill) error {
	// add bill to db logic
	// need to make idempotent!!!
	// Simulating DB logic (replace with actual SQL)
	fmt.Printf("Saving bill %s to DB\n", bill.ID)
	return nil
}
