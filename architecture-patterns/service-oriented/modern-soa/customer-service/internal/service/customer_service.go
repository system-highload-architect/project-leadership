package service

import (
	"errors"
)

type CustomerService struct{}

func NewCustomerService() *CustomerService {
	return &CustomerService{}
}

func (s *CustomerService) GetCustomer(id string) (map[string]interface{}, error) {
	if id == "" {
		return nil, errors.New("customer id is required")
	}
	return map[string]interface{}{
		"id":   id,
		"name": "John Doe",
		"city": "Moscow",
	}, nil
}
