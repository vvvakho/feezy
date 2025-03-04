package users

//encore:service
type Service struct {
}

func initService() (*Service, error) {
	return &Service{}, nil
}
