package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"eats/backend/orders/adapters/db/dbmodels"
	"eats/backend/orders/api/http"
)

// ReadModel provides read-optimized queries that return HTTP response types directly.
type ReadModel struct {
	db *pgxpool.Pool
}

func NewReadModel(db *pgxpool.Pool) *ReadModel {
	if db == nil {
		panic("db connection pool cannot be nil")
	}
	return &ReadModel{db: db}
}

// TODO: Implement this method using sqlc to query menu items joined with restaurants.
func (r ReadModel) ListMenuItemsWithRestaurant(ctx context.Context, params http.ListMenuItemsFilter) ([]http.MenuItemWithRestaurant, error) {
	queries := dbmodels.New(r.db)

	result, err := queries.ListMenuItemsWithRestaurant(ctx,
		dbmodels.ListMenuItemsWithRestaurantParams{
			RestaurantName: params.RestaurantName,
			OrderBy:        params.OrderBy,
		},
	)
	if err != nil {
		return nil, err
	}
	var menuItemsWithRestaurant []http.MenuItemWithRestaurant
	for _, item := range result {
		menuItemsWithRestaurant = append(menuItemsWithRestaurant, http.MenuItemWithRestaurant{
			MenuItemUuid:   item.OrdersRestaurantMenuItem.RestaurantMenuItemUuid,
			MenuItemName:   item.OrdersRestaurantMenuItem.Name,
			GrossPrice:     item.OrdersRestaurantMenuItem.GrossPrice,
			RestaurantUuid: item.OrdersRestaurant.RestaurantUuid,
			RestaurantName: item.OrdersRestaurant.Name,
			Currency:       item.OrdersRestaurant.Currency,
		})
	}

	return menuItemsWithRestaurant, nil
}
