// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
//go:build integration

package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"eats/backend/common"
	"eats/backend/common/shared"
	"eats/backend/common/testutils"
	"eats/backend/orders/adapters/db"
	"eats/backend/orders/adapters/db/dbmodels"
	"eats/backend/orders/app"
)

func TestCreateQuote(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPool := testutils.NewDB(t)
	ordersRepo := db.NewOrdersRepository(dbPool)
	restaurantRepo := db.NewRestaurantRepository(dbPool)
	customerRepo := db.NewCustomerRepository(dbPool)

	// Setup: Create a restaurant with menu positions
	restaurantUUID := app.RestaurantUUID{common.NewUUIDv7()}
	restaurant := newTestOnboardRestaurant()
	err := restaurantRepo.UpsertRestaurant(ctx, restaurantUUID, restaurant)
	require.NoError(t, err)

	// Create a customer
	customerUUID := app.CustomerUUID{common.NewUUIDv7()}
	customer := newTestCustomer(customerUUID)
	err = customerRepo.RegisterCustomer(ctx, customer)
	require.NoError(t, err)

	deliveryAddress := testutils.GenerateRandomAddress(testutils.GenerateRandomCountry())

	quoteMenuItems := []app.CreateQuoteItem{
		{MenuItemUUID: restaurant.MenuItems[0].MenuItemUUID, Quantity: 2},
		{MenuItemUUID: restaurant.MenuItems[1].MenuItemUUID, Quantity: 1},
	}

	var createdQuote app.Quote
	var positions []app.QuoteMenuItem
	quote, err := ordersRepo.CreateQuote(ctx, restaurantUUID, quoteMenuItems, func(
		ctx context.Context,
		menuItems map[app.RestaurantMenuItemUUID]app.MenuItem,
		r app.Restaurant,
	) (app.Quote, []app.QuoteMenuItem, error) {
		// Verify menu positions were fetched correctly
		require.Len(t, menuItems, 2)
		if diff := cmp.Diff(
			app.Restaurant{
				RestaurantUUID: restaurantUUID,
				Name:           restaurant.Name,
				Description:    restaurant.Description,
				Address:        restaurant.Address,
				Currency:       restaurant.Currency,
			},
			r,
			cmpopts.EquateComparable(shared.SharedTypes...),
		); diff != "" {
			t.Errorf("restaurant mismatch (-want +got):\n%s", diff)
		}

		// Calculate totals
		itemsSubtotal := decimal.Zero
		positions = make([]app.QuoteMenuItem, 0, len(quoteMenuItems))

		for _, qmp := range quoteMenuItems {
			mp, ok := menuItems[qmp.MenuItemUUID]
			require.True(t, ok)
			itemsSubtotal = itemsSubtotal.Add(mp.GrossPrice.Mul(decimal.NewFromInt(int64(qmp.Quantity))))

			positions = append(positions, app.QuoteMenuItem{
				MenuItemUUID: qmp.MenuItemUUID,
				GrossPrice:   mp.GrossPrice,
				Quantity:     qmp.Quantity,
			})
		}

		serviceFee := decimal.NewFromFloat(5.00)
		deliveryFee := decimal.NewFromFloat(3.00)
		totalAmount := itemsSubtotal.Add(serviceFee).Add(deliveryFee)
		totalTax := totalAmount.Mul(decimal.NewFromFloat(0.1)).RoundBank(2)

		createdQuote = app.Quote{
			QuoteUUID:          app.QuoteUUID{common.NewUUIDv7()},
			CustomerUUID:       customerUUID,
			RestaurantUUID:     restaurantUUID,
			DeliveryAddress:    deliveryAddress,
			ItemsSubtotalGross: itemsSubtotal,
			ServiceFeeGross:    serviceFee,
			DeliveryFeeGross:   deliveryFee,
			TotalAmountGross:   totalAmount,
			TotalTax:           totalTax,
			Currency:           r.Currency,
			CreatedAt:          time.Now(),
		}

		return createdQuote, positions, nil
	})

	require.NoError(t, err)

	cmpOpts := cmp.Options{
		cmpopts.EquateComparable(shared.SharedTypes...),
		cmp.Comparer(func(a, b time.Time) bool {
			return a.Truncate(time.Second).Equal(b.Truncate(time.Second))
		}),
		cmp.Comparer(func(a, b decimal.Decimal) bool {
			return a.Equal(b)
		}),
	}

	// Verify the returned quote matches what we created
	if diff := cmp.Diff(createdQuote, quote, cmpOpts); diff != "" {
		t.Errorf("quote mismatch (-want +got):\n%s", diff)
	}

	// Verify the quote was persisted correctly
	queries := dbmodels.New(dbPool)
	persistedQuote, err := queries.GetQuote(ctx, quote.QuoteUUID)
	require.NoError(t, err)

	persistedAsAppQuote := app.Quote{
		QuoteUUID:          persistedQuote.QuoteUuid,
		CustomerUUID:       persistedQuote.CustomerUuid,
		RestaurantUUID:     persistedQuote.RestaurantUuid,
		DeliveryAddress:    persistedQuote.DeliveryAddress,
		ItemsSubtotalGross: persistedQuote.ItemsSubtotalGross,
		ServiceFeeGross:    persistedQuote.ServiceFeeGross,
		DeliveryFeeGross:   persistedQuote.DeliveryFeeGross,
		TotalAmountGross:   persistedQuote.TotalAmountGross,
		TotalTax:           persistedQuote.TotalTax,
		Currency:           persistedQuote.Currency,
		CreatedAt:          persistedQuote.CreatedAt,
	}
	if diff := cmp.Diff(createdQuote, persistedAsAppQuote, cmpOpts); diff != "" {
		t.Errorf("persisted quote mismatch (-want +got):\n%s", diff)
	}

	// Verify quote items were persisted correctly
	quoteItems, err := queries.GetQuoteItems(ctx, quote.QuoteUUID)
	require.NoError(t, err)
	require.Len(t, quoteItems, len(positions))

	persistedPositions := make([]app.QuoteMenuItem, 0, len(quoteItems))
	for _, item := range quoteItems {
		persistedPositions = append(persistedPositions, app.QuoteMenuItem{
			MenuItemUUID: item.MenuItemUuid,
			GrossPrice:   item.GrossPrice,
			Quantity:     int(item.Quantity),
		})
	}
	if diff := cmp.Diff(positions, persistedPositions, cmpOpts, cmpopts.SortSlices(func(a, b app.QuoteMenuItem) bool {
		return a.MenuItemUUID.String() < b.MenuItemUUID.String()
	})); diff != "" {
		t.Errorf("persisted quote items mismatch (-want +got):\n%s", diff)
	}
}

func TestQuoteWithMenuItems(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPool := testutils.NewDB(t)
	ordersRepo := db.NewOrdersRepository(dbPool)
	restaurantRepo := db.NewRestaurantRepository(dbPool)
	customerRepo := db.NewCustomerRepository(dbPool)

	// Setup: Create restaurant and quote
	_, quote, quoteMenuItems := setupRestaurantAndQuote(t, ctx, restaurantRepo, ordersRepo, customerRepo)

	// Fetch quote with menu items from the repository
	fetchedQuote, menuItems, err := ordersRepo.QuoteWithMenuItems(ctx, quote.QuoteUUID)
	require.NoError(t, err)

	// Verify the quote matches
	if diff := cmp.Diff(
		quote,
		fetchedQuote,
		cmpopts.EquateComparable(shared.SharedTypes...),
		cmp.Comparer(func(a, b time.Time) bool {
			return a.Truncate(time.Second).Equal(b.Truncate(time.Second))
		}),
	); diff != "" {
		t.Errorf("quote mismatch (-want +got):\n%s", diff)
	}

	// Verify menu items are returned
	require.Len(t, menuItems, len(quoteMenuItems))
	for _, qmi := range quoteMenuItems {
		_, ok := menuItems[qmi.MenuItemUUID]
		assert.True(t, ok, "menu item %s not found in returned items", qmi.MenuItemUUID)
	}
}

func TestSaveOrder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPool := testutils.NewDB(t)
	ordersRepo := db.NewOrdersRepository(dbPool)
	restaurantRepo := db.NewRestaurantRepository(dbPool)
	customerRepo := db.NewCustomerRepository(dbPool)

	// Setup: Create restaurant and quote
	_, quote, _ := setupRestaurantAndQuote(t, ctx, restaurantRepo, ordersRepo, customerRepo)

	// Create order from quote
	order, err := app.NewOrderFromQuote(quote)
	require.NoError(t, err)

	// Save order
	err = ordersRepo.SaveOrder(ctx, order)
	require.NoError(t, err)
}

func setupRestaurantAndQuote(t *testing.T, ctx context.Context, restaurantRepo *db.RestaurantRepository, ordersRepo *db.OrdersRepo, customerRepo *db.CustomerRepository) (app.RestaurantUUID, app.Quote, []app.CreateQuoteItem) {
	// Create restaurant
	restaurantUUID := app.RestaurantUUID{common.NewUUIDv7()}
	restaurant := newTestOnboardRestaurant()
	err := restaurantRepo.UpsertRestaurant(ctx, restaurantUUID, restaurant)
	require.NoError(t, err)

	// Create customer
	customerUUID := app.CustomerUUID{common.NewUUIDv7()}
	customer := newTestCustomer(customerUUID)
	err = customerRepo.RegisterCustomer(ctx, customer)
	require.NoError(t, err)

	deliveryAddress := testutils.GenerateRandomAddress(testutils.GenerateRandomCountry())
	quoteMenuItems := []app.CreateQuoteItem{
		{MenuItemUUID: restaurant.MenuItems[0].MenuItemUUID, Quantity: 1},
	}

	quote, err := ordersRepo.CreateQuote(ctx, restaurantUUID, quoteMenuItems, func(
		ctx context.Context,
		menuItems map[app.RestaurantMenuItemUUID]app.MenuItem,
		r app.Restaurant,
	) (app.Quote, []app.QuoteMenuItem, error) {
		itemsSubtotal := decimal.Zero
		quoteItems := make([]app.QuoteMenuItem, 0, len(quoteMenuItems))

		for _, qmp := range quoteMenuItems {
			mp := menuItems[qmp.MenuItemUUID]
			itemsSubtotal = itemsSubtotal.Add(mp.GrossPrice.Mul(decimal.NewFromInt(int64(qmp.Quantity))))

			quoteItems = append(quoteItems, app.QuoteMenuItem{
				MenuItemUUID: qmp.MenuItemUUID,
				Quantity:     qmp.Quantity,
				GrossPrice:   mp.GrossPrice,
			})
		}

		serviceFee := decimal.NewFromFloat(5.00)
		deliveryFee := decimal.NewFromFloat(3.00)
		totalAmount := itemsSubtotal.Add(serviceFee).Add(deliveryFee)
		totalTax := totalAmount.Mul(decimal.NewFromFloat(0.1)).RoundBank(2)

		return app.Quote{
			QuoteUUID:          app.QuoteUUID{common.NewUUIDv7()},
			CustomerUUID:       customerUUID,
			RestaurantUUID:     restaurantUUID,
			DeliveryAddress:    deliveryAddress,
			ItemsSubtotalGross: itemsSubtotal,
			ServiceFeeGross:    serviceFee,
			DeliveryFeeGross:   deliveryFee,
			TotalAmountGross:   totalAmount,
			TotalTax:           totalTax,
			Currency:           r.Currency,
			CreatedAt:          time.Now(),
		}, quoteItems, nil
	})
	require.NoError(t, err)

	return restaurantUUID, quote, quoteMenuItems
}
