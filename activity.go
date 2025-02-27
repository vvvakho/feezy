package fees

import "golang.org/x/net/context"

type Service struct {
	db string
}

func (s *Service) AddBillToDB(ctx context.Context, bill BillState) error {
	// add bill to db logic
	// need to make idempotent!!!
	return nil
}
