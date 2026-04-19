package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"eats/backend/common"
	"eats/backend/common/shared"
	"eats/backend/orders/adapters/db/dbmodels"
	"eats/backend/orders/api/http"
)

type CustomerRepository struct {
	db *pgxpool.Pool
}

func NewCustomerRepository(db *pgxpool.Pool) *CustomerRepository {
	if db == nil {
		panic("db connection pool cannot be nil")
	}

	return &CustomerRepository{
		db: db,
	}
}

func (r *CustomerRepository) RegisterCustomer(ctx context.Context, customerUUID common.UUID, customer http.RegisterCustomer) error {
	query := dbmodels.New(r.db)
	address, err := shared.NewAddress(
		customer.Address.Line1,
		customer.Address.Line2,
		customer.Address.PostalCode,
		customer.Address.City,
		customer.Address.CountryCode,
	)

	email := string(customer.Email)
	err = query.InsertCustomer(ctx, dbmodels.InsertCustomerParams{
		CustomerUuid: customerUUID,
		Name:         customer.Name,
		Email:        email,
		Address:      address,
		PhoneNumber:  customer.PhoneNumber,
	})
	if err != nil {
		return err

	}
	return nil
}
