package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"eats/backend/common"
	"eats/backend/orders/adapters/db/dbmodels"
	"eats/backend/orders/app"
)

type OrdersRepo struct {
	db *pgxpool.Pool
}

func NewOrdersRepository(db *pgxpool.Pool) *OrdersRepo {
	if db == nil {
		panic("db connection pool cannot be nil")
	}

	return &OrdersRepo{db: db}
}

func (r *OrdersRepo) GetRestaurant(
	ctx context.Context,
	restaurantID app.RestaurantUUID,
) (app.Restaurant, error) {
	queries := dbmodels.New(r.db)
	dbRestaurant, err := queries.GetRestaurant(ctx, restaurantID)
	if err != nil {
		return app.Restaurant{}, fmt.Errorf("failed to get restaurant %s: %w", restaurantID, err)
	}
	return appRestaurantFromDB(dbRestaurant), nil
}

func (r *OrdersRepo) CreateQuote(
	ctx context.Context,
	restaurantID app.RestaurantUUID,
	menuItems app.CreateQuoteItems,
	updateFn func(
		ctx context.Context,
		menuItems map[app.RestaurantMenuItemUUID]app.MenuItem,
		restaurant app.Restaurant,
	) (app.Quote, []app.QuoteMenuItem, error),
) (app.Quote, error) {
	var quote app.Quote

	err := common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		queries := dbmodels.New(tx)

		menuItemsUUIDs := make([]common.UUID, 0, len(menuItems))
		for _, item := range menuItems {
			menuItemsUUIDs = append(menuItemsUUIDs, item.MenuItemUUID.UUID)
		}

		appMenuItems, err := r.getMenuItems(ctx, queries, restaurantID, menuItemsUUIDs)
		if err != nil {
			return err
		}

		dbRestaurant, err := queries.GetRestaurant(ctx, restaurantID)
		if err != nil {
			return fmt.Errorf("failed to get restaurant currency for restaurant %s: %w", restaurantID, err)
		}

		var items []app.QuoteMenuItem
		quote, items, err = updateFn(ctx, appMenuItems, appRestaurantFromDB(dbRestaurant))
		if err != nil {
			return fmt.Errorf("failed to create quote using updateFn: %w", err)
		}

		err = queries.AddQuote(ctx, dbmodels.AddQuoteParams{
			QuoteUuid:          quote.QuoteUUID,
			CustomerUuid:       quote.CustomerUUID,
			RestaurantUuid:     quote.RestaurantUUID,
			DeliveryAddress:    quote.DeliveryAddress,
			ItemsSubtotalGross: quote.ItemsSubtotalGross,
			ServiceFeeGross:    quote.ServiceFeeGross,
			DeliveryFeeGross:   quote.DeliveryFeeGross,
			TotalAmountGross:   quote.TotalAmountGross,
			TotalTax:           quote.TotalTax,
			CreatedAt:          time.Now(),
			Currency:           quote.Currency,
		})
		if err != nil {
			return fmt.Errorf("failed to add quote %s: %w", quote.QuoteUUID, err)
		}

		quoteItems := dbQuoteItemsFromApp(items, quote)

		if _, err := queries.AddQuoteItems(ctx, quoteItems); err != nil {
			return fmt.Errorf("failed to add quote items for quote %s: %w", quote.QuoteUUID, err)
		}

		return nil
	})
	if err != nil {
		return app.Quote{}, err
	}

	return quote, nil
}

func dbQuoteItemsFromApp(menuItems []app.QuoteMenuItem, quote app.Quote) []dbmodels.AddQuoteItemsParams {
	quoteItems := make([]dbmodels.AddQuoteItemsParams, 0, len(menuItems))
	for _, position := range menuItems {
		quoteItems = append(quoteItems, dbmodels.AddQuoteItemsParams{
			QuoteItemUuid: common.NewUUIDv7(),
			QuoteUuid:     quote.QuoteUUID,
			MenuItemUuid:  position.MenuItemUUID,
			GrossPrice:    position.GrossPrice,
			Quantity:      int32(position.Quantity),
		})
	}
	return quoteItems
}

func (r *OrdersRepo) getMenuItems(ctx context.Context, queries *dbmodels.Queries, restaurantUUID app.RestaurantUUID, menuItemsUUIDs []common.UUID) (map[app.RestaurantMenuItemUUID]app.MenuItem, error) {
	dbMenuItems, err := queries.GetMenuItemsByUUIDs(ctx, dbmodels.GetMenuItemsByUUIDsParams{
		RestaurantUuid: restaurantUUID,
		Column2:        menuItemsUUIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get menu positions: %w", err)
	}

	return appMenuItemsFromDbMenuItems(dbMenuItems), nil
}

func appMenuItemsFromDbMenuItems(dbMenuItems []dbmodels.OrdersRestaurantMenuItem) map[app.RestaurantMenuItemUUID]app.MenuItem {
	appMenuItems := make(map[app.RestaurantMenuItemUUID]app.MenuItem, len(dbMenuItems))

	for _, dbItemPosition := range dbMenuItems {
		appMenuItems[dbItemPosition.RestaurantMenuItemUuid] = app.MenuItem{
			MenuItemUUID: dbItemPosition.RestaurantMenuItemUuid,
			Name:         dbItemPosition.Name,
			Ordering:     dbItemPosition.Ordering,
			GrossPrice:   dbItemPosition.GrossPrice,
			IsArchived:   dbItemPosition.IsArchived,
		}
	}

	return appMenuItems
}

func appRestaurantFromDB(r dbmodels.OrdersRestaurant) app.Restaurant {
	return app.Restaurant{
		RestaurantUUID: r.RestaurantUuid,
		Name:           r.Name,
		Description:    r.Description,
		Address:        r.Address,
		Currency:       r.Currency,
	}
}

func (r *OrdersRepo) QuoteWithMenuItems(ctx context.Context, quoteUUID app.QuoteUUID) (app.Quote, map[app.RestaurantMenuItemUUID]app.MenuItem, error) {
	queries := dbmodels.New(r.db)

	dbQuote, err := queries.GetQuote(ctx, quoteUUID)
	if err != nil {
		return app.Quote{}, nil, fmt.Errorf("failed to get quote %s: %w", quoteUUID, err)
	}

	dbPositions, err := queries.GetMenuItemsForQuote(ctx, quoteUUID)
	if err != nil {
		return app.Quote{}, nil, fmt.Errorf("failed to get menu positions for quote %s: %w", quoteUUID, err)
	}

	return appQuoteFromDbQuote(dbQuote), appMenuItemsFromDbMenuItems(dbPositions), nil
}

func (r *OrdersRepo) SaveOrder(ctx context.Context, order app.Order) error {
	return common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		queries := dbmodels.New(tx)

		err := queries.AddOrder(ctx, dbOrderFromAppOrder(order))
		if err != nil {
			return fmt.Errorf("failed to add order %s: %w", order.OrderUUID, err)
		}

		return nil
	})
}

func (r *OrdersRepo) UpdateOrder(ctx context.Context,
	orderUUID app.OrderUUID,
	updateFn func(ctx context.Context, order app.Order) (app.Order, error),
) error {
	return common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		queries := dbmodels.New(tx)

		dbOrder, err := queries.GetOrder(ctx, orderUUID)
		if err != nil {
			return err
		}

		updatedOrder, err := updateFn(ctx, dbOrderToAppOrder(dbOrder))
		if err != nil {
			return fmt.Errorf("update function failed for order %s: %w", orderUUID, err)
		}
		return queries.UpdateOrder(ctx, dbmodels.UpdateOrderParams{
			OrderUuid:             orderUUID,
			CourierUuid:           updatedOrder.CourierUUID,
			OrderedAt:             &updatedOrder.OrderedAt,
			RestaurantConfirmedAt: updatedOrder.RestaurantConfirmedAt,
			CourierAcceptedAt:     updatedOrder.CourierAcceptedAt,
			RestaurantPreparedAt:  updatedOrder.RestaurantPreparedAt,
			PickedUpAt:            updatedOrder.PickedUpAt,
			DeliveredAt:           updatedOrder.DeliveredAt,
		})
	})
}

func (r *OrdersRepo) GetOrder(ctx context.Context, orderUUID app.OrderUUID) (app.Order, error) {
	queries := dbmodels.New(r.db)

	dbOrder, err := queries.GetOrder(ctx, orderUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.Order{}, common.NewNotFoundError(
			"order_not_found",
			"order with UUID %s not found",
			orderUUID,
		)
	}
	if err != nil {
		return app.Order{}, fmt.Errorf("failed to get order %s: %w", orderUUID, err)
	}

	return dbOrderToAppOrder(dbOrder), nil
}

func dbOrderToAppOrder(dbOrder dbmodels.OrdersOrder) app.Order {
	return app.Order{
		OrderUUID:             dbOrder.OrderUuid,
		QuoteUUID:             dbOrder.QuoteUuid,
		CustomerUUID:          dbOrder.CustomerUuid,
		RestaurantUUID:        dbOrder.RestaurantUuid,
		CourierUUID:           dbOrder.CourierUuid,
		DeliveryAddress:       dbOrder.DeliveryAddress,
		OrderedAt:             dbOrder.OrderedAt,
		RestaurantConfirmedAt: dbOrder.RestaurantConfirmedAt,
		CourierAcceptedAt:     dbOrder.CourierAcceptedAt,
		RestaurantPreparedAt:  dbOrder.RestaurantPreparedAt,
		PickedUpAt:            dbOrder.PickedUpAt,
		DeliveredAt:           dbOrder.DeliveredAt,
		ItemsSubtotal:         dbOrder.ItemsSubtotalGross,
		ServiceFeeGross:       dbOrder.ServiceFeeGross,
		DeliveryFeeGross:      dbOrder.DeliveryFeeGross,
		TotalAmountGross:      dbOrder.TotalAmountGross,
		TotalTax:              dbOrder.TotalTax,
		Currency:              dbOrder.Currency,
	}
}

func dbOrderFromAppOrder(order app.Order) dbmodels.AddOrderParams {
	return dbmodels.AddOrderParams{
		OrderUuid:          order.OrderUUID,
		QuoteUuid:          order.QuoteUUID,
		CustomerUuid:       order.CustomerUUID,
		RestaurantUuid:     order.RestaurantUUID,
		DeliveryAddress:    order.DeliveryAddress,
		ItemsSubtotalGross: order.ItemsSubtotal,
		ServiceFeeGross:    order.ServiceFeeGross,
		DeliveryFeeGross:   order.DeliveryFeeGross,
		TotalAmountGross:   order.TotalAmountGross,
		TotalTax:           order.TotalTax,
		OrderedAt:          order.OrderedAt,
		Currency:           order.Currency,
	}
}

func appQuoteFromDbQuote(dbQuote dbmodels.OrdersQuote) app.Quote {
	return app.Quote{
		QuoteUUID:          dbQuote.QuoteUuid,
		CustomerUUID:       dbQuote.CustomerUuid,
		RestaurantUUID:     dbQuote.RestaurantUuid,
		DeliveryAddress:    dbQuote.DeliveryAddress,
		ItemsSubtotalGross: dbQuote.ItemsSubtotalGross,
		ServiceFeeGross:    dbQuote.ServiceFeeGross,
		DeliveryFeeGross:   dbQuote.DeliveryFeeGross,
		TotalAmountGross:   dbQuote.TotalAmountGross,
		TotalTax:           dbQuote.TotalTax,
		Currency:           dbQuote.Currency,
		CreatedAt:          dbQuote.CreatedAt,
	}
}
