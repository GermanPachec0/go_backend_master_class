package http

import (
	"context"
	"fmt"

	"eats/backend/common"
	"eats/backend/common/shared"
	"eats/backend/orders/app"
)

type Handler struct {
	service *app.Service
}

func NewHandler(
	service *app.Service,
) Handler {
	if service == nil {
		panic("service cannot be nil")
	}

	return Handler{
		service: service,
	}
}

func (h Handler) RegisterCustomer(ctx context.Context, request RegisterCustomerRequestObject) (RegisterCustomerResponseObject, error) {
	addr, err := openapiAddressToSharedAddress(request.Body.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	customerUUID := common.NewUUIDv7()

	err = h.service.RegisterCustomer(ctx, app.Customer{
		CustomerUUID: app.CustomerUUID{UUID: customerUUID},
		Name:         request.Body.Name,
		Email:        string(request.Body.Email),
		// address should be ideally normalized to ensure consistent city names and postal codes
		// across customers, restaurants, and delivery addresses
		Address:     addr,
		PhoneNumber: request.Body.PhoneNumber,
	})
	if err != nil {
		return nil, err
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
