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

func (r ReadModel) ListCustomerOrders(ctx context.Context, customerUUID app.CustomerUUID) ([]http.CustomerOrder, error) {
	queries := dbmodels.New(r.db)

	rows, err := queries.ListCustomerOrders(ctx, customerUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query customer orders: %w", err)
	}

	orders := make([]http.CustomerOrder, 0, len(rows))
	for _, row := range rows {
		restaurantName, err := queries.GetRestaurantName(ctx, row.RestaurantUuid)
		if err != nil {
			return nil, fmt.Errorf("failed to query restaurant name for order %s: %w", row.OrderUuid.String(), err)
		}

		deliveryAddress := http.Address{
			CountryCode: row.DeliveryAddress.CountryCode,
			City:        row.DeliveryAddress.City,
			Line1:       row.DeliveryAddress.Line1,
			Line2:       row.DeliveryAddress.Line2,
			PostalCode:  row.DeliveryAddress.PostalCode,
		}

		orders = append(orders, http.CustomerOrder{
			CourierAcceptedAt:     row.CourierAcceptedAt,
			CourierUuid:           row.CourierUuid,
			Currency:              row.Currency,
			DeliveredAt:           row.DeliveredAt,
			DeliveryAddress:       deliveryAddress,
			DeliveryFeeGross:      row.DeliveryFeeGross,
			ItemsSubtotalGross:    row.ItemsSubtotalGross,
			OrderUuid:             row.OrderUuid,
			OrderedAt:             row.OrderedAt,
			PickedUpAt:            row.PickedUpAt,
			RestaurantConfirmedAt: row.RestaurantConfirmedAt,
			RestaurantName:        restaurantName,
			RestaurantPreparedAt:  row.RestaurantPreparedAt,
			RestaurantUuid:        row.RestaurantUuid,
			ServiceFeeGross:       row.ServiceFeeGross,
			TotalGross:            row.TotalAmountGross,
			TotalTax:              row.TotalTax,
		})
	}

	return orders, nil
}

func (r ReadModel) ListRestaurantOrders(ctx context.Context, restaurantUUID app.RestaurantUUID) ([]http.RestaurantOrder, error) {
	queries := dbmodels.New(r.db)

	rows, err := queries.ListRestaurantOrders(ctx, restaurantUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query restaurant orders: %w", err)
	}

	orders := make([]http.RestaurantOrder, 0, len(rows))
	for _, row := range rows {
		orders = append(orders, http.RestaurantOrder{
			CourierAcceptedAt:     row.CourierAcceptedAt,
			CourierUuid:           row.CourierUuid,
			DeliveredAt:           row.DeliveredAt,
			ItemsSubtotalGross:    row.ItemsSubtotalGross,
			OrderUuid:             row.OrderUuid,
			OrderedAt:             row.OrderedAt,
			PickedUpAt:            row.PickedUpAt,
			RestaurantConfirmedAt: row.RestaurantConfirmedAt,
			RestaurantPreparedAt:  row.RestaurantPreparedAt,
			CustomerUuid:          row.CustomerUuid,
		})
	}

	return orders, nil
}

func (r ReadModel) ListAssignedCourierOrders(ctx context.Context, courierUUID app.CourierUUID) ([]http.CourierOrder, error) {
	queries := dbmodels.New(r.db)

	rows, err := queries.ListAssignedCourierOrders(ctx, &courierUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query assigned courier orders: %w", err)
	}

	orders := make([]http.CourierOrder, 0, len(rows))
	for _, row := range rows {
		orders = append(orders, http.CourierOrder{
			AcceptedByCourierAt:   row.CourierAcceptedAt,
			DeliveredAt:           row.DeliveredAt,
			OrderUuid:             row.OrderUuid,
			OrderedAt:             row.OrderedAt,
			PickedUpAt:            row.PickedUpAt,
			RestaurantConfirmedAt: row.RestaurantConfirmedAt,
			RestaurantPreparedAt:  row.RestaurantPreparedAt,
			CustomerUuid:          row.CustomerUuid,
			RestaurantUuid:        row.RestaurantUuid,
			CourierUuid:           row.CourierUuid,
			DeliveryAddress: http.Address{
				City:        row.DeliveryAddress.City,
				CountryCode: row.DeliveryAddress.CountryCode,
				Line1:       row.DeliveryAddress.Line1,
				Line2:       row.DeliveryAddress.Line2,
				PostalCode:  row.DeliveryAddress.PostalCode,
			},
			ItemsSubtotalGross: row.ItemsSubtotalGross,
			RestaurantName:     row.RestaurantName,
		})
	}

	return orders, nil
}

func (r ReadModel) ListAvailableOrdersForCourier(ctx context.Context, courierUUID app.CourierUUID) ([]http.CourierOrder, error) {
	queries := dbmodels.New(r.db)

	rows, err := queries.ListAvailableOrdersForCourier(ctx, courierUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query available orders for courier: %w", err)
	}

	orders := make([]http.CourierOrder, 0, len(rows))
	for _, row := range rows {
		orders = append(orders, http.CourierOrder{
			OrderUuid:             row.OrderUuid,
			OrderedAt:             row.OrderedAt,
			RestaurantConfirmedAt: row.RestaurantConfirmedAt,
			RestaurantPreparedAt:  row.RestaurantPreparedAt,
			CustomerUuid:          row.CustomerUuid,
			RestaurantUuid:        row.RestaurantUuid,
			RestaurantName:        row.RestaurantName,
			AcceptedByCourierAt:   row.CourierAcceptedAt,
			CourierUuid:           row.CourierUuid,
			DeliveredAt:           row.DeliveredAt,
			PickedUpAt:            row.PickedUpAt,
			ItemsSubtotalGross:    row.ItemsSubtotalGross,
			DeliveryAddress: http.Address{
				City:        row.DeliveryAddress.City,
				CountryCode: row.DeliveryAddress.CountryCode,
				Line1:       row.DeliveryAddress.Line1,
				Line2:       row.DeliveryAddress.Line2,
				PostalCode:  row.DeliveryAddress.PostalCode,
			},
		})
	}

	return orders, nil
}
