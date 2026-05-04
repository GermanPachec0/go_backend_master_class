// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package tests_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"eats/backend/common/testutils"
	"eats/backend/orders/api/http/client"
)

func TestComponent_CriticalFlow(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)

	ctx := t.Context()

	country := testutils.GenerateRandomCountry()

	customerUUID := registerCustomerInCity(ctx, t, clients, country, "Some city")
	assert.NotEmpty(t, customerUUID)
}

func TestComponent_ListMenuItems(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)

	ctx := t.Context()
	country := testutils.GenerateRandomCountry()

	// Onboard a restaurant with menu items
	restaurantUUID, menuItems := onboardRestaurant(ctx, t, clients, country, "Test Restaurant")
	require.NotEmpty(t, restaurantUUID)
	require.NotEmpty(t, menuItems)

	// Call the read model endpoint (no filters)
	resp, err := clients.Orders.ListMenuItemsWithResponse(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotNil(t, resp.JSON200)

	// Verify our menu items are in the response
	items := *resp.JSON200
	found := 0
	for _, item := range items {
		for _, expected := range menuItems {
			if item.MenuItemUuid == expected.Uuid {
				assert.Equal(t, expected.Name, item.MenuItemName)
				assert.Equal(t, "Test Restaurant", item.RestaurantName)
				found++
			}
		}
	}
	assert.Equal(t, len(menuItems), found, "all menu items should be returned by read model")
}

func TestComponent_ListMenuItems_WithFiltering(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)

	ctx := t.Context()
	country := testutils.GenerateRandomCountry()

	// Onboard two restaurants
	_, _ = onboardRestaurant(ctx, t, clients, country, "Pizza Palace")
	_, _ = onboardRestaurant(ctx, t, clients, country, "Burger Barn")

	// Filter by restaurant name
	restaurantName := "Pizza"
	resp, err := clients.Orders.ListMenuItemsWithResponse(ctx, &client.ListMenuItemsParams{
		RestaurantName: &restaurantName,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotNil(t, resp.JSON200)

	// All items should be from Pizza Palace
	items := *resp.JSON200
	for _, item := range items {
		assert.Contains(t, item.RestaurantName, "Pizza", "all items should be from filtered restaurant")
	}
}

func TestComponent_ListMenuItems_WithOrdering(t *testing.T) {
	t.Parallel()
	clients := newTestClients(t)

	ctx := t.Context()
	country := testutils.GenerateRandomCountry()

	// Onboard a restaurant
	_, _ = onboardRestaurant(ctx, t, clients, country, "Test Restaurant")

	// Order by price ascending
	orderBy := client.PriceAsc
	resp, err := clients.Orders.ListMenuItemsWithResponse(ctx, &client.ListMenuItemsParams{
		OrderBy: &orderBy,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotNil(t, resp.JSON200)

	// Verify items are ordered by price
	items := *resp.JSON200
	if len(items) > 1 {
		for i := 1; i < len(items); i++ {
			assert.True(t, items[i-1].GrossPrice.LessThanOrEqual(items[i].GrossPrice),
				"items should be ordered by price ascending")
		}
	}
}
