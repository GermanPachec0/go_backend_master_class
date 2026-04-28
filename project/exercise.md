# Integrate Quotes Service

The last step to make creating quotes work is exposing a new HTTP endpoint (`POST /orders/customer/create-quote`).
The [OpenAPI](https://academy.threedots.tech/knowledge/openapi) spec is already updated.

The HTTP handler should work like `OnboardRestaurant` in `backend/orders/api/http/handler.go`.
It should map OpenAPI request types to app types, call the service, and map the response back.
You've done this with `RegisterCustomer` and `OnboardRestaurant`, so the pattern should feel familiar.

The customer UUID comes from `request.Params.CustomerUUID` now.
It's a header parameter coming from the authenticated user calling the endpoint.

The handler should return `CustomerCreateQuote201JSONResponse` with fields mapped from the `Quote` app type.
Most are direct field copies, but expiration uses the `quote.ExpirationTime()` method.

## Exercise

Exercise path: ./project

1. **Run `task gen` (or `go generate ./...`)** to get the new OpenAPI types.
2. **Implement `CustomerCreateQuote`** on `Handler` in `backend/orders/api/http/handler.go`.
3. Map `request.Body.Items` to `[]app.CreateQuoteItem` and convert the delivery address as you did in the other endpoints.
4. Build the `app.CreateQuote` struct, call `h.service.CreateQuote`, and return a `CustomerCreateQuote201JSONResponse` with all the quote fields.

The platform will verify that creating a quote returns the correct fields.

{{hints}}

{{hint 1}}

The handler signature and the items mapping can look like this:

```go
func (h Handler) CustomerCreateQuote(ctx context.Context, request CustomerCreateQuoteRequestObject) (CustomerCreateQuoteResponseObject, error) {
    var items []app.CreateQuoteItem
    for _, item := range request.Body.Items {
        items = append(items, app.CreateQuoteItem{
            MenuItemUUID: item.MenuItemUuid,
            Quantity:     item.Quantity,
        })
    }

    // ...
}
```

{{endhint}}

{{endhints}}
