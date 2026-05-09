package db

import (
	"context"
	"fmt"

	"eats/backend/common"
	"eats/backend/orders/adapters/db/dbmodels"
	"eats/backend/orders/app"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CourierRepository struct {
	db *pgxpool.Pool
}

func NewCourierRepository(db *pgxpool.Pool) *CourierRepository {
	return &CourierRepository{db: db}
}

func (r *CourierRepository) RegisterCourier(ctx context.Context, courier app.Courier) (app.CourierUUID, error) {
	err := common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		queries := dbmodels.New(tx)

		err := queries.InsertCourier(ctx, dbmodels.InsertCourierParams{
			CourierUuid: courier.CourierUUID.UUID,
			Name:        courier.Name,
			PhoneNumber: courier.PhoneNumber,
			City:        courier.City,
		})
		if err != nil {
			return fmt.Errorf("insert courier failed: %w", err)
		}

		return nil
	})
	if err != nil {
		return app.CourierUUID{}, err
	}
	return courier.CourierUUID, nil
}
