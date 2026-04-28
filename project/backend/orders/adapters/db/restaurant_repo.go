package db

import (
	"context"
	"errors"
	"fmt"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"eats/backend/common"
	"eats/backend/common/log"
	"eats/backend/orders/adapters/db/dbmodels"
	"eats/backend/orders/app"
)

type RestaurantRepository struct {
	db *pgxpool.Pool
}

func NewRestaurantRepository(db *pgxpool.Pool) *RestaurantRepository {
	if db == nil {
		panic("db connection pool cannot be nil")
	}

	return &RestaurantRepository{
		db: db,
	}
}

func (r *RestaurantRepository) UpsertRestaurant(ctx context.Context, restaurantUUID app.RestaurantUUID, restaurant app.OnboardRestaurant) error {
	queries := dbmodels.New(r.db)

	log.FromContext(ctx).With("restaurant_uuid", restaurantUUID).Info("Upserting restaurant")

	dbRestaurant, err := queries.UpsertRestaurant(ctx, dbmodels.UpsertRestaurantParams{
		restaurantUUID,
		restaurant.Name,
		restaurant.Description,
		restaurant.Address,
		restaurant.Currency,
	})
	if err != nil {
		return fmt.Errorf("upsert restaurant failed: %w", err)
	}

	// Currency is immutable after creation - the upsert doesn't update it.
	// Check here catches attempts to change it and returns a clear error.
	if dbRestaurant.Currency != restaurant.Currency {
		return common.NewInvalidInputError("cannot-change-currency", "cannot change restaurant currency once set")
	}

	// TODO: upsert menu items

	return nil
}

func (r *RestaurantRepository) GetRestaurantMenu(ctx context.Context, restaurantUUID app.RestaurantUUID) (app.RestaurantMenu, error) {
	queries := dbmodels.New(r.db)

	dbItems, err := queries.GetRestaurantMenu(ctx, restaurantUUID)
	if err != nil {
		return app.RestaurantMenu{}, fmt.Errorf("get restaurant menu failed: %w", err)
	}

	log.FromContext(ctx).With("restaurant_uuid", restaurantUUID, "count", len(dbItems)).Info("Fetched menu items")

	items := make([]app.MenuItem, len(dbItems))
	for i, dbItem := range dbItems {
		items[i] = app.MenuItem{
			MenuItemUUID: dbItem.OrdersRestaurantMenuItem.RestaurantMenuItemUuid,
			Name:         dbItem.OrdersRestaurantMenuItem.Name,
			GrossPrice:   dbItem.OrdersRestaurantMenuItem.GrossPrice,
			Ordering:     dbItem.OrdersRestaurantMenuItem.Ordering,
		}
	}

	restaurant, err := queries.GetRestaurant(ctx, restaurantUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return app.RestaurantMenu{}, common.NewNotFoundError("restaurant-not-found", "restaurant not found")
		}
		return app.RestaurantMenu{}, fmt.Errorf("get restaurant %s failed: %w", restaurantUUID, err)
	}

	return app.RestaurantMenu{
		RestaurantName: restaurant.Name,
		Address:        restaurant.Address,
		Description:    restaurant.Description,
		Currency:       restaurant.Currency,
		Positions:      items,
	}, nil
}
