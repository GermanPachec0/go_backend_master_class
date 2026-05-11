package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"eats/backend/orders/adapters/db/dbmodels"
	"eats/backend/orders/api/http"
	"eats/backend/orders/app"
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

func (r ReadModel) ListRestaurants(ctx context.Context) ([]http.Restaurant, error) {
	queries := dbmodels.New(r.db)

	rows, err := queries.ListRestaurants(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query restaurants: %w", err)
	}

	restaurants := make([]http.Restaurant, 0, len(rows))
	for _, row := range rows {

		address := http.Address{
			CountryCode: row.Address.CountryCode,
			City:        row.Address.City,
			Line1:       row.Address.Line1,
			Line2:       row.Address.Line2,
			PostalCode:  row.Address.PostalCode,
		}
		restaurants = append(restaurants, http.Restaurant{
			Uuid:        row.RestaurantUuid,
			Name:        row.Name,
			Description: row.Description,
			Address:     address,
			Currency:    row.Currency,
		})
	}

	return restaurants, nil
}

func (r ReadModel) CustomerGetRestaurantMenu(ctx context.Context, restaurantUUID app.RestaurantUUID) ([]http.MenuItem, error) {
	queries := dbmodels.New(r.db)

	rows, err := queries.GetRestaurantMenu(ctx, restaurantUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query restaurant menu: %w", err)
	}

	menuItems := make([]http.MenuItem, 0, len(rows))
	for _, row := range rows {
		menuItems = append(menuItems, http.MenuItem{
			Uuid:       row.OrdersRestaurantMenuItem.RestaurantMenuItemUuid,
			Name:       row.OrdersRestaurantMenuItem.Name,
			GrossPrice: row.OrdersRestaurantMenuItem.GrossPrice,
			Ordering:   float32(row.OrdersRestaurantMenuItem.Ordering),
		})
	}

	return menuItems, nil
}

func (r ReadModel) CustomerGetRestaurant(ctx context.Context, restaurantUUID app.RestaurantUUID) (http.Restaurant, error) {
	queries := dbmodels.New(r.db)

	row, err := queries.GetRestaurant(ctx, restaurantUUID)
	if err != nil {
		return http.Restaurant{}, fmt.Errorf("failed to query restaurant: %w", err)
	}

	address := http.Address{
		CountryCode: row.Address.CountryCode,
		City:        row.Address.City,
		Line1:       row.Address.Line1,
		Line2:       row.Address.Line2,
		PostalCode:  row.Address.PostalCode,
	}

	return http.Restaurant{
		Uuid:        row.RestaurantUuid,
		Name:        row.Name,
		Description: row.Description,
		Address:     address,
		Currency:    row.Currency,
	}, nil
}
