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

	customerUUID := CustomerUUID{common.NewUUIDv7()}

	err = h.service.RegisterCustomer(ctx, app.Customer{
		CustomerUUID: customerUUID,
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

func (h Handler) OnboardRestaurant(ctx context.Context, request OnboardRestaurantRequestObject) (OnboardRestaurantResponseObject, error) {
	if request.Params.OperatorUUID.IsZero() {
		return nil, common.NewUnauthorizedError("missing-operator-uuid", "operator UUID is required")
	}

	var menuItems []app.MenuItem
	for _, item := range request.Body.MenuItems {
		menuItems = append(menuItems, app.MenuItem{
			MenuItemUUID: item.Uuid,
			Name:         item.Name,
			GrossPrice:   item.GrossPrice,
			Ordering:     float64(item.Ordering),
		})
	}

	addr, err := openapiAddressToSharedAddress(request.Body.Address)
	if err != nil {
		return nil, common.NewInvalidInputError("invalid-address", "invalid address: %s", err)
	}

	err = h.service.OnboardRestaurant(
		ctx,
		request.RestaurantUuid,
		app.OnboardRestaurant{
			request.Body.Name,
			addr,
			request.Body.Currency,
			request.Body.Description,
			menuItems,
		},
	)
	if err != nil {
		return nil, err
	}

	return OnboardRestaurant204Response{}, nil
}

func (h Handler) CustomerCreateQuote(ctx context.Context, request CustomerCreateQuoteRequestObject) (CustomerCreateQuoteResponseObject, error) {
	var quoteItems []app.CreateQuoteItem
	for _, items := range request.Body.Items {
		quoteItems = append(quoteItems, app.CreateQuoteItem{
			MenuItemUUID: items.MenuItemUuid,
			Quantity:     items.Quantity,
		})
	}

	address, err := openapiAddressToSharedAddress(request.Body.DeliveryAddress)
	if err != nil {
		return nil, common.NewInvalidInputError("invalid-address", "invalid address: %s", err)
	}

	quote, err := h.service.CreateQuote(ctx, app.CreateQuote{
		RestaurantUUID:  request.Body.RestaurantUuid,
		CustomerUUID:    request.Params.CustomerUUID,
		DeliveryAddress: address,
		QuoteItems:      quoteItems,
	})
	if err != nil {
		return nil, err
	}

	return CustomerCreateQuote201JSONResponse{
		QuoteUuid: quote.QuoteUUID,
	}, nil
}

func Register(ctx context.Context, e EchoRouter, handler Handler) error {
	RegisterHandlers(e, NewStrictHandler(handler, nil))

	return nil
}
