package db

import (
	"context"
	"fmt"

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

func (r ReadModel) ListMenuItemsWithRestaurant(ctx context.Context, filter http.ListMenuItemsFilter) ([]http.MenuItemWithRestaurant, error) {
	queries := dbmodels.New(r.db)

	rows, err := queries.ListMenuItemsWithRestaurant(ctx, dbmodels.ListMenuItemsWithRestaurantParams{
		SearchTerm:           filter.Search,
		RestaurantNameFilter: filter.RestaurantName,
		OrderBy:              filter.OrderBy,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query menu items: %w", err)
	}

	// Map directly to HTTP response types - no domain objects needed for reads
	items := make([]http.MenuItemWithRestaurant, 0, len(rows))
	for _, row := range rows {
		items = append(items, http.MenuItemWithRestaurant{
			MenuItemUuid:   row.MenuItemUuid,
			MenuItemName:   row.MenuItemName,
			GrossPrice:     row.GrossPrice,
			Currency:       row.Currency,
			RestaurantUuid: row.RestaurantUuid,
			RestaurantName: row.RestaurantName,
		})
	}

	return items, nil
}
