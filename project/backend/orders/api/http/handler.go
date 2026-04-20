package http

import (
	"context"

	"eats/backend/common"
	"eats/backend/common/shared"
	"eats/backend/orders/app"
)

type CustomerRepository interface {
	RegisterCustomer(ctx context.Context, customer app.Customer) error
}
type Handler struct {
	svc CustomerRepository
}

func NewHandler(
	svc CustomerRepository,
) Handler {
	if svc == nil {
		panic("svc cannot be nil")
	}

	return Handler{
		svc: svc,
	}
}

func (h Handler) RegisterCustomer(ctx context.Context, request RegisterCustomerRequestObject) (RegisterCustomerResponseObject, error) {
	customerUUID := common.NewUUIDv7()

	address, err := openapiAddressToSharedAddress(request.Body.Address)
	if err != nil {
		return nil, err
	}

	customer := app.Customer{
		UUID:        customerUUID,
		Name:        request.Body.Name,
		Email:       string(request.Body.Email),
		Address:     address,
		PhoneNumber: request.Body.PhoneNumber,
	}

	err = h.svc.RegisterCustomer(ctx, customer)
	if err != nil {
		return nil, err
	}

	return RegisterCustomer201JSONResponse{
		CustomerUuid: customerUUID,
	}, nil
}

func Register(ctx context.Context, e EchoRouter, handler Handler) error {
	RegisterHandlers(e, NewStrictHandler(handler, nil))

	return nil
}

func openapiAddressToSharedAddress(addr Address) (shared.Address, error) {
	return shared.Address{
		City:        addr.City,
		CountryCode: shared.CountryCode(addr.CountryCode),
		Line1:       addr.Line1,
		Line2:       addr.Line2,
		PostalCode:  addr.PostalCode,
	}, nil
}
