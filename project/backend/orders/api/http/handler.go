package http

import (
	"context"

	"eats/backend/common"
	"eats/backend/common/shared"
)

type CustomerRepository interface {
	RegisterCustomer(ctx context.Context, customerUUID common.UUID, customer RegisterCustomer) error
}

type Handler struct {
	CustomerRepository CustomerRepository
}

func NewHandler(
	customerRepository CustomerRepository,
) Handler {
	if customerRepository == nil {
		panic("customerRepository cannot be nil")
	}

	return Handler{
		CustomerRepository: customerRepository,
	}
}

func (h Handler) RegisterCustomer(ctx context.Context, request RegisterCustomerRequestObject) (RegisterCustomerResponseObject, error) {
	customer := request.Body
	customerUUID := common.NewUUIDv7()

	err := h.CustomerRepository.RegisterCustomer(ctx, customerUUID, *customer)

	if err != nil {
		return RegisterCustomer409JSONResponse{
			Message: err.Error(),
		}, nil
	}

	return RegisterCustomer201JSONResponse{
		CustomerUuid: customerUUID,
	}, nil
}

func openapiAddressToSharedAddress(addr Address) (shared.Address, error) {
	sharedAddr, err := shared.NewAddress(
		addr.Line1,
		addr.Line2,
		addr.PostalCode,
		addr.City,
		addr.CountryCode,
	)
	if err != nil {
		return shared.Address{}, err
	}

	return sharedAddr, nil
}

func Register(ctx context.Context, e EchoRouter, handler Handler) error {
	RegisterHandlers(e, NewStrictHandler(handler, nil))

	return nil
}
