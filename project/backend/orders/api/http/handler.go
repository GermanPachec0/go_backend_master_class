package http

import (
	"context"

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
		return nil, common.NewInvalidInputError("invalid-address", "invalid address: %s", err)
	}

	customerUUID := common.NewUUIDv7()

	err = h.service.RegisterCustomer(ctx, app.Customer{
		CustomerUUID: app.CustomerUUID{UUID: customerUUID},
		Name:         request.Body.Name,
		Email:        string(request.Body.Email),
		Address:      addr,
		PhoneNumber:  request.Body.PhoneNumber,
	})
	if err != nil {
		return nil, err
	}

	return RegisterCustomer201JSONResponse{
		CustomerUuid: app.CustomerUUID{UUID: customerUUID},
	}, nil
}

func (h Handler) OnboardRestaurant(ctx context.Context, request OnboardRestaurantRequestObject) (OnboardRestaurantResponseObject, error) {
	if request.Params.OperatorUUID.IsZero() {
		return nil, common.NewUnauthorizedError("missing-operator-uuid", "operator UUID is required")
	}

	var menuItems []app.MenuItem
	for _, item := range request.Body.MenuItems {
		newItem := app.MenuItem{
			Name:         item.Name,
			MenuItemUUID: item.Uuid,
			Ordering:     float64(item.Ordering),
			GrossPrice:   item.GrossPrice,
		}
		menuItems = append(menuItems, newItem)
	}

	addr, err := openapiAddressToSharedAddress(request.Body.Address)
	if err != nil {
		return nil, common.NewInvalidInputError("invalid-address", "invalid address: %s", err)
	}

	err = h.service.OnboardRestaurant(ctx,
		request.RestaurantUuid,
		app.OnboardRestaurant{
			Name:        request.Body.Name,
			Description: request.Body.Description,
			Currency:    request.Body.Currency,
			Address:     addr,
			MenuItems:   menuItems,
		})
	if err != nil {
		return nil, err
	}

	return OnboardRestaurant204Response{}, nil
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
