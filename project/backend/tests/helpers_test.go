// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package tests_test

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"testing"
	"time"

	bank2 "github.com/ThreeDotsLabs/the-domain-engineer/clients/bank"
	gofakeit "github.com/brianvoe/gofakeit/v7"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"eats/backend/common"
	"eats/backend/common/shared"
	"eats/backend/common/testutils"
	ordersclient "eats/backend/orders/api/http/client"
	"eats/backend/orders/app"
)

type testRestaurant struct {
	UUID app.RestaurantUUID
	Data ordersclient.OnboardRestaurant
}

func onboardRestaurant(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	country shared.CountryCode,
) testRestaurant {
	t.Helper()

	var menuItems []ordersclient.MenuItem
	for i := 0; i < 5; i++ {
		menuItems = append(menuItems, ordersclient.MenuItem{
			Uuid:       app.RestaurantMenuItemUUID{common.NewUUIDv7()},
			Name:       gofakeit.Lunch(),
			GrossPrice: randomPrice(),
			Ordering:   rand.Float32(),
		})
	}

	name := ""
	if rand.Intn(2) == 0 {
		name += gofakeit.FirstName() + "'s "
	}
	name += gofakeit.HipsterWord()

	address := testutils.GenerateRandomOpenapiAddress(country)

	restaurantToCreate := ordersclient.OnboardRestaurant{
		Address:     address,
		Description: gofakeit.HipsterSentence(),
		MenuItems:   menuItems,
		Name:        cases.Title(language.Und).String(name),
		Currency:    currencyForCountry(t, country),
	}

	restaurantUUID := app.RestaurantUUID{common.NewUUIDv7()}
	resp, err := clients.Orders.OnboardRestaurantWithResponse(
		ctx,
		restaurantUUID,
		&ordersclient.OnboardRestaurantParams{
			OperatorUUID: common.NewUUIDv7(),
		},
		restaurantToCreate,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode())

	return testRestaurant{
		UUID: restaurantUUID,
		Data: restaurantToCreate,
	}
}

func registerCustomer(ctx context.Context, t *testing.T, clients testClients, country shared.CountryCode) ordersclient.CustomerUUID {
	t.Helper()

	customerToCreate := ordersclient.RegisterCustomer{
		Name:        gofakeit.Name(),
		Email:       openapi_types.Email(gofakeit.Email()),
		Address:     testutils.GenerateRandomOpenapiAddress(country),
		PhoneNumber: gofakeit.Phone(),
	}

	resp, err := clients.Orders.RegisterCustomerWithResponse(ctx, customerToCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())
	require.NotNil(t, resp.JSON201)

	return resp.JSON201.CustomerUuid
}

func registerCustomerInCity(ctx context.Context, t *testing.T, clients testClients, country shared.CountryCode, city string) ordersclient.CustomerUUID {
	t.Helper()

	customerToCreate := ordersclient.RegisterCustomer{
		Name:        gofakeit.Name(),
		Email:       openapi_types.Email(gofakeit.Email()),
		Address:     testutils.GenerateOpenapiAddressInCity(country, city),
		PhoneNumber: gofakeit.Phone(),
	}

	resp, err := clients.Orders.RegisterCustomerWithResponse(ctx, customerToCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())
	require.NotNil(t, resp.JSON201)

	return resp.JSON201.CustomerUuid
}

type testCourier struct {
	UUID ordersclient.CourierUUID
}

func registerCourierInCity(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	country shared.CountryCode,
	city string,
) testCourier {
	t.Helper()

	courierToCreate := ordersclient.RegisterCourier{
		Name:        gofakeit.Name(),
		PhoneNumber: gofakeit.Phone(),
		City:        city,
	}

	resp, err := clients.Orders.RegisterCourierWithResponse(ctx, courierToCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())
	require.NotNil(t, resp.JSON201)

	return testCourier{
		UUID: resp.JSON201.CourierUuid,
	}
}

func restaurantAcceptOrder(ctx context.Context, t *testing.T, clients testClients, restaurantUUID app.RestaurantUUID, orderUUID app.OrderUUID) {
	t.Helper()

	resp, err := clients.Orders.RestaurantAcceptOrderWithResponse(
		ctx,
		&ordersclient.RestaurantAcceptOrderParams{
			RestaurantUUID: restaurantUUID,
		},
		ordersclient.AcceptOrder{
			OrderUuid: orderUUID,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode())
}

func courierAcceptDelivery(ctx context.Context, t *testing.T, clients testClients, courierUUID app.CourierUUID, orderUUID app.OrderUUID) {
	t.Helper()

	resp, err := clients.Orders.CourierAcceptDeliveryWithResponse(
		ctx,
		&ordersclient.CourierAcceptDeliveryParams{
			CourierUUID: courierUUID,
		},
		ordersclient.AcceptDelivery{
			OrderUuid: orderUUID,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode())
}

func restaurantMarkOrderReady(ctx context.Context, t *testing.T, clients testClients, restaurantUUID app.RestaurantUUID, orderUUID app.OrderUUID) {
	t.Helper()

	resp, err := clients.Orders.RestaurantMarkOrderReadyForPickupWithResponse(
		ctx,
		&ordersclient.RestaurantMarkOrderReadyForPickupParams{
			RestaurantUUID: restaurantUUID,
		},
		ordersclient.MarkOrderReady{
			OrderUuid: orderUUID,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode())
}

func courierReportPickup(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	courierUUID app.CourierUUID,
	orderUUID app.OrderUUID,
) {
	t.Helper()

	resp, err := clients.Orders.CourierReportPickupWithResponse(
		ctx,
		&ordersclient.CourierReportPickupParams{
			CourierUUID: courierUUID,
		},
		ordersclient.ReportPickup{
			OrderUuid: orderUUID,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode())
}

func courierReportDelivered(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	courierUUID app.CourierUUID,
	orderUUID app.OrderUUID,
) {
	t.Helper()

	resp, err := clients.Orders.CourierReportDeliveryWithResponse(
		ctx,
		&ordersclient.CourierReportDeliveryParams{
			CourierUUID: courierUUID,
		},
		ordersclient.ReportDelivery{
			OrderUuid: orderUUID,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode())
}

func updateRestaurantMenu(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	restaurantUUID app.RestaurantUUID,
	restaurant ordersclient.OnboardRestaurant,
) {
	t.Helper()

	resp, err := clients.Orders.OnboardRestaurantWithResponse(
		ctx,
		restaurantUUID,
		&ordersclient.OnboardRestaurantParams{
			OperatorUUID: common.NewUUIDv7(),
		},
		restaurant,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode())
}

func randomPrice() decimal.Decimal {
	return decimal.New(int64(rand.Intn(200)+50), -1)
}

func onboardRestaurantWithData(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	restaurantUUID app.RestaurantUUID,
	restaurant ordersclient.OnboardRestaurant,
) {
	t.Helper()

	resp, err := clients.Orders.OnboardRestaurantWithResponse(
		ctx,
		restaurantUUID,
		&ordersclient.OnboardRestaurantParams{
			OperatorUUID: common.NewUUIDv7(),
		},
		restaurant,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode())
}

func createQuote(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	customerUUID app.CustomerUUID,
	restaurantUUID app.RestaurantUUID,
	orderItems []ordersclient.OrderItem,
	deliveryAddress ordersclient.Address,
) *ordersclient.CreateQuoteResponse {
	t.Helper()

	createQuoteRequest := ordersclient.CreateQuoteRequest{
		RestaurantUuid:  restaurantUUID,
		Items:           orderItems,
		DeliveryAddress: deliveryAddress,
	}

	resp, err := clients.Orders.CustomerCreateQuoteWithResponse(
		ctx,
		&ordersclient.CustomerCreateQuoteParams{
			CustomerUUID: customerUUID,
		},
		createQuoteRequest,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())
	require.NotNil(t, resp.JSON201)

	return resp.JSON201
}

func placeOrderFromQuote(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	customerUUID app.CustomerUUID,
	restaurantUUID app.RestaurantUUID,
	quote *ordersclient.CreateQuoteResponse,
) *ordersclient.CustomerOrder {
	t.Helper()

	_, cardNumber := createBankAccountWithBalance(ctx, t, clients, decimal.NewFromInt(1000), common.NewUUIDv7().String())
	createBankAccount(ctx, t, clients, restaurantUUID.String())
	nonce := preauthPayment(ctx, t, clients, cardNumber, quote.TotalGross, quote.Currency.String(), quote.QuoteUuid.String())

	placeOrderRequest := ordersclient.PlaceOrder{
		QuoteUuid:    quote.QuoteUuid,
		PaymentNonce: nonce,
	}

	resp, err := clients.Orders.CustomerPlaceOrderWithResponse(
		ctx,
		&ordersclient.CustomerPlaceOrderParams{
			CustomerUUID: customerUUID,
		},
		placeOrderRequest,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())
	require.NotNil(t, resp.JSON201)

	return resp.JSON201
}

func assertOrderMatchesQuote(t *testing.T, order *ordersclient.CustomerOrder, quote *ordersclient.CreateQuoteResponse) {
	t.Helper()

	assert.Equal(
		t,
		quote.ItemsSubtotalGross.String(),
		order.ItemsSubtotalGross.String(),
		"order items subtotal should match quote",
	)
	assert.Equal(
		t,
		quote.ServiceFeeGross.String(),
		order.ServiceFeeGross.String(),
		"order service fee should match quote",
	)
	assert.Equal(
		t,
		quote.DeliveryFeeGross.String(),
		order.DeliveryFeeGross.String(),
		"order delivery fee should match quote",
	)
	assert.Equal(
		t,
		quote.TotalGross.String(),
		order.TotalGross.String(),
		"order total gross should match quote",
	)
	assert.Equal(
		t,
		quote.TotalTax.String(),
		order.TotalTax.String(),
		"order total tax should match quote",
	)
}

func currencyForCountry(t *testing.T, country shared.CountryCode) shared.Currency {
	t.Helper()
	switch country.Code() {
	case "US":
		return shared.MustNewCurrency("USD")
	case "DE":
		return shared.MustNewCurrency("EUR")
	case "GB":
		return shared.MustNewCurrency("GBP")
	case "JP":
		return shared.MustNewCurrency("JPY")
	case "PL":
		return shared.MustNewCurrency("PLN")
	default:
		t.Fatalf("unsupported country for currency mapping: %s", country.Code())
		return shared.Currency{} // unreachable
	}
}

func assertJsonReprEqual(t *testing.T, expected, actual any) {
	t.Helper()

	expectedJSON, err := json.Marshal(expected)
	require.NoError(t, err)

	actualJSON, err := json.Marshal(actual)
	require.NoError(t, err)

	require.JSONEq(t, string(expectedJSON), string(actualJSON))
}

func createBankAccount(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	merchantID string,
) (string, string) {
	t.Helper()
	return createBankAccountWithBalance(ctx, t, clients, decimal.Zero, merchantID)
}

func createBankAccountWithBalance(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	balance decimal.Decimal,
	merchantID string,
) (string, string) {
	t.Helper()
	resp, err := clients.CommonClients.Bank.CreateAccountWithResponse(ctx, bank2.CreateAccountJSONRequestBody{
		InitialBalance: balance,
		MerchantId:     merchantID,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode())
	require.NotNil(t, resp.JSON201)
	require.NotNil(t, resp.JSON201.Card)
	return resp.JSON201.AccountNumber, resp.JSON201.Card.CardNumber
}

func preauthPayment(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	cardNumber string,
	amount decimal.Decimal,
	currency string,
	idempotencyKey string,
) string {
	paymentResp, err := clients.CommonClients.Bank.PreauthorizePaymentWithResponse(ctx, bank2.PreauthorizePaymentJSONRequestBody{
		Amount:     amount,
		CardNumber: cardNumber,
		Currency:   currency,
		Cvv:        "123",
		ExpiryDate: openapi_types.Date{
			Time: time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		IdempotencyKey: idempotencyKey,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, paymentResp.StatusCode())

	return paymentResp.JSON200.PaymentNonce
}

func onboardRestaurantWithName(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	country shared.CountryCode,
	name string,
) (app.RestaurantUUID, ordersclient.OnboardRestaurant) {
	t.Helper()

	var menuItems []ordersclient.MenuItem
	for i := 0; i < 5; i++ {
		menuItems = append(menuItems, ordersclient.MenuItem{
			Uuid:       app.RestaurantMenuItemUUID{common.NewUUIDv7()},
			Name:       gofakeit.Lunch(),
			GrossPrice: randomPrice(),
			Ordering:   rand.Float32(),
		})
	}

	restaurantToCreate := ordersclient.OnboardRestaurant{
		Address:     testutils.GenerateRandomOpenapiAddress(country),
		Description: gofakeit.HipsterSentence(),
		MenuItems:   menuItems,
		Name:        name,
		Currency:    currencyForCountry(t, country),
	}

	restaurantUUID := app.RestaurantUUID{common.NewUUIDv7()}
	resp, err := clients.Orders.OnboardRestaurantWithResponse(
		ctx,
		restaurantUUID,
		&ordersclient.OnboardRestaurantParams{
			OperatorUUID: common.NewUUIDv7(),
		},
		restaurantToCreate,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode())

	return restaurantUUID, restaurantToCreate
}

func onboardRestaurantWithItems(
	ctx context.Context,
	t *testing.T,
	clients testClients,
	country shared.CountryCode,
	name string,
	itemNames []string,
) (app.RestaurantUUID, ordersclient.OnboardRestaurant) {
	t.Helper()

	menuItems := make([]ordersclient.MenuItem, 0, len(itemNames))
	for i, itemName := range itemNames {
		menuItems = append(menuItems, ordersclient.MenuItem{
			Uuid:       app.RestaurantMenuItemUUID{common.NewUUIDv7()},
			Name:       itemName,
			GrossPrice: decimal.NewFromFloat(10.00 + float64(i)),
			Ordering:   float32(i + 1),
		})
	}

	restaurantToCreate := ordersclient.OnboardRestaurant{
		Address:     testutils.GenerateRandomOpenapiAddress(country),
		Description: gofakeit.HipsterSentence(),
		MenuItems:   menuItems,
		Name:        name,
		Currency:    currencyForCountry(t, country),
	}

	restaurantUUID := app.RestaurantUUID{common.NewUUIDv7()}
	resp, err := clients.Orders.OnboardRestaurantWithResponse(
		ctx,
		restaurantUUID,
		&ordersclient.OnboardRestaurantParams{
			OperatorUUID: common.NewUUIDv7(),
		},
		restaurantToCreate,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode())

	return restaurantUUID, restaurantToCreate
}
