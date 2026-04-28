package db

import (
	"context"
	"fmt"

	"eats/backend/common"
	"eats/backend/common/shared"
	"eats/backend/orders/adapters/db/dbmodels"
	"eats/backend/orders/app"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrdersRepository struct {
	db *pgxpool.Pool
}

func NewOrdersRepository(db *pgxpool.Pool) *OrdersRepository {
	if db == nil {
		panic("db connection pool cannot be nil")
	}

	return &OrdersRepository{
		db: db,
	}
}

func (r *OrdersRepository) CreateQuote(
	ctx context.Context,
	restaurantUUID app.RestaurantUUID,
	menuItems app.CreateQuoteItems,
	updateFn func(
		ctx context.Context,
		menuItems map[app.RestaurantMenuItemUUID]app.MenuItem,
		restaurantCurrency shared.Currency,
		restaurantAddress shared.Address,
	) (app.Quote, []app.QuoteMenuItem, error),
) (app.Quote, error) {
	var finalQuote app.Quote
	err := common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
		queries := dbmodels.New(tx)

		menuItems, err := queries.GetMenuItemsByUUIDs(ctx, dbmodels.GetMenuItemsByUUIDsParams{
			RestaurantUuid: restaurantUUID,
			MenuItemUuids:  menuItems.MenuItemUUIDs(),
		})
		if err != nil {
			return err
		}

		restaurant, err := queries.GetRestaurant(ctx, restaurantUUID)
		if err != nil {
			return err
		}

		menuItemsMap := make(map[app.RestaurantMenuItemUUID]app.MenuItem)
		for _, item := range menuItems {
			menuItemsMap[app.RestaurantMenuItemUUID(item.OrdersRestaurantMenuItem.RestaurantMenuItemUuid)] = app.MenuItem{
				MenuItemUUID: app.RestaurantMenuItemUUID(item.OrdersRestaurantMenuItem.RestaurantMenuItemUuid),
				Name:         item.OrdersRestaurantMenuItem.Name,
				Ordering:     item.OrdersRestaurantMenuItem.Ordering,
				GrossPrice:   item.OrdersRestaurantMenuItem.GrossPrice,
				IsArchived:   item.OrdersRestaurantMenuItem.IsArchived,
			}
		}

		quote, quote_items, err := updateFn(
			ctx,
			menuItemsMap,
			shared.Currency(restaurant.Currency),
			shared.Address(restaurant.Address),
		)
		if err != nil {
			return err
		}

		dbQuote, err := queries.AddQuote(ctx, dbmodels.AddQuoteParams{
			QuoteUuid:          quote.QuoteUUID,
			CustomerUuid:       quote.CustomerUUID,
			RestaurantUuid:     quote.RestaurantUUID,
			DeliveryAddress:    quote.DeliveryAddress,
			ItemsSubtotalGross: quote.ItemsSubtotalGross,
			Currency:           restaurant.Currency,
			ServiceFeeGross:    quote.ServiceFeeGross,
			DeliveryFeeGross:   quote.DeliveryFeeGross,
			TotalAmountGross:   quote.TotalAmountGross,
			TotalTax:           quote.TotalTax,
			CreatedAt:          quote.CreatedAt,
		})
		if err != nil {
			return err
		}

		finalQuote = app.Quote{
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

		var quoteItemsParams []dbmodels.AddQuoteItemsParams
		for _, item := range quote_items {
			quoteItemsParams = append(quoteItemsParams, dbmodels.AddQuoteItemsParams{
				QuoteUuid:    quote.QuoteUUID,
				MenuItemUuid: item.MenuItemUUID,
				Quantity:     int32(item.Quantity),
				GrossPrice:   item.GrossPrice,
			})
		}

		quantity_items, err := queries.AddQuoteItems(ctx, quoteItemsParams)
		if err != nil {
			return err
		}

		if quantity_items != int64(len(quote_items)) {
			return fmt.Errorf("quantity items added are not equal to items")
		}

		return nil
	})

	return finalQuote, err
}
