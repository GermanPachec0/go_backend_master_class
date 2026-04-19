package http

import (
	"context"

	"eats/backend/common"
	"eats/backend/common/shared"
	"eats/backend/orders/adapters/db/dbmodels"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) Handler {
	return Handler{db: db}
}

func (h Handler) RegisterCustomer(ctx context.Context, request RegisterCustomerRequestObject) (RegisterCustomerResponseObject, error) {
	customerUUID := common.NewUUIDv7()
	query := dbmodels.New(h.db)
	address, err := shared.NewAddress(
		request.Body.Address.Line1,
		request.Body.Address.Line2,
		request.Body.Address.PostalCode,
		request.Body.Address.City,
		request.Body.Address.CountryCode,
	)
	if err != nil {
		return RegisterCustomer400JSONResponse{
			BadRequestJSONResponse: BadRequestJSONResponse{
				Message: "failed to create address",
			},
		}, nil
	}
	email := string(request.Body.Email)

	err = query.InsertCustomer(ctx, dbmodels.InsertCustomerParams{
		CustomerUuid: customerUUID,
		Name:         request.Body.Name,
		Email:        email,
		Address:      address,
		PhoneNumber:  request.Body.PhoneNumber,
	})
	if err != nil {
		return RegisterCustomer400JSONResponse{
			BadRequestJSONResponse: BadRequestJSONResponse{
				Message: "failed to register customer",
			},
		}, nil
	}
	return RegisterCustomer201JSONResponse{
		CustomerUuid: customerUUID,
	}, nil
}

func Register(ctx context.Context, e EchoRouter, handler Handler) error {
	RegisterHandlers(e, NewStrictHandler(handler, nil))

	return nil
}
