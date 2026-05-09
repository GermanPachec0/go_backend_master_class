package db

import (
	"context"
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
			quote.QuoteUUID,
			quote.CustomerUUID,
			quote.RestaurantUUID,
			quote.DeliveryAddress,
			quote.ItemsSubtotalGross,
			quote.ServiceFeeGross,
			quote.DeliveryFeeGross,
			quote.TotalAmountGross,
			quote.TotalTax,
			time.Now(),
			quote.Currency,
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

func (r *OrdersRepo) GetQuote(ctx context.Context, quoteUUID app.QuoteUUID) (app.Quote, error) {
	queries := dbmodels.New(r.db)
	dbQuote, err := queries.GetQuote(ctx, quoteUUID)
	if err != nil {
		return app.Quote{}, fmt.Errorf("failed to get quote %s: %w", quoteUUID, err)
	}
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
	}, nil
}

func (r *OrdersRepo) GetMenuItemsForQuote(ctx context.Context, quoteUUID app.QuoteUUID) ([]app.MenuItem, error) {
	queries := dbmodels.New(r.db)
	dbMenuItems, err := queries.GetMenuItemsForQuote(ctx, quoteUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get menu items for quote %s: %w", quoteUUID, err)
	}

	menuItems := make([]app.MenuItem, 0, len(dbMenuItems))
	for _, dbItem := range dbMenuItems {
		menuItems = append(menuItems, app.MenuItem{
			MenuItemUUID: dbItem.RestaurantMenuItemUuid,
			Name:         dbItem.Name,
			GrossPrice:   dbItem.GrossPrice,
			Ordering:     dbItem.Ordering,
			IsArchived:   dbItem.IsArchived,
		})
	}

	return menuItems, nil
}

func (r *OrdersRepo) PlaceOrder(ctx context.Context, quote app.Quote) (app.Order, error) {
	var order app.Order
	err := common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		queries := dbmodels.New(tx)
		orderUUID := app.OrderUUID{common.NewUUIDv7()}

		err := queries.InsertOrder(ctx, dbmodels.InsertOrderParams{
			OrderUuid:          orderUUID,
			QuoteUuid:          quote.QuoteUUID,
			CustomerUuid:       quote.CustomerUUID,
			RestaurantUuid:     quote.RestaurantUUID,
			DeliveryAddress:    quote.DeliveryAddress,
			ItemsSubtotalGross: quote.ItemsSubtotalGross,
			ServiceFeeGross:    quote.ServiceFeeGross,
			DeliveryFeeGross:   quote.DeliveryFeeGross,
			TotalAmountGross:   quote.TotalAmountGross,
			TotalTax:           quote.TotalTax,
			CourierUuid:        nil, // Assuming courier UUID is not available at this point
			Currency:           quote.Currency,
		})
		if err != nil {
			return fmt.Errorf("insert order failed: %w", err)
		}

		order = app.Order{
			OrderUUID:          orderUUID,
			QuoteUUID:          quote.QuoteUUID,
			CustomerUUID:       quote.CustomerUUID,
			RestaurantUUID:     quote.RestaurantUUID,
			DeliveryAddress:    quote.DeliveryAddress,
			ItemsSubtotalGross: quote.ItemsSubtotalGross,
			ServiceFeeGross:    quote.ServiceFeeGross,
			DeliveryFeeGross:   quote.DeliveryFeeGross,
			TotalAmountGross:   quote.TotalAmountGross,
			TotalTax:           quote.TotalTax,
			CourierUUID:        nil, // Assuming courier UUID is not available at this point
			Currency:           quote.Currency,
		}

		return nil
	})

	if err != nil {
		return app.Order{}, err
	}

	return order, nil
}

func dbQuoteItemsFromApp(menuItems []app.QuoteMenuItem, quote app.Quote) []dbmodels.AddQuoteItemsParams {
	quoteItems := make([]dbmodels.AddQuoteItemsParams, 0, len(menuItems))
	for _, position := range menuItems {
		quoteItems = append(quoteItems, dbmodels.AddQuoteItemsParams{
			common.NewUUIDv7(),
			quote.QuoteUUID,
			position.MenuItemUUID,
			position.GrossPrice,
			int32(position.Quantity),
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
			dbItemPosition.RestaurantMenuItemUuid,
			dbItemPosition.Name,
			dbItemPosition.Ordering,
			dbItemPosition.GrossPrice,
			dbItemPosition.IsArchived,
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
