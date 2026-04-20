package app

import (
	"context"
)

type CustomerRepository interface {
	RegisterCustomer(ctx context.Context, customer Customer) error
}

type Service struct {
	customerRepository CustomerRepository
}

func NewService(customerRepository CustomerRepository) *Service {
	if customerRepository == nil {
		panic("customerRepository cannot be nil")
	}

	return &Service{
		customerRepository: customerRepository,
	}
}

func (s *Service) RegisterCustomer(ctx context.Context, customer Customer) error {
	return s.customerRepository.RegisterCustomer(ctx, customer)
}
