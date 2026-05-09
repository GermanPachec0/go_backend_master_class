# Place Order

So far, our project made changes to a single database. This allowed us to wrap many operations in a single transaction. If one failed, we rolled back all of them.

Now we're adding support for placing orders, and things get tricky. We need to call the payments API to charge the customer's credit card. This is an external system, so we can't keep it within a transaction.

We covered why in {{exerciseLink "Delivery Service" "10-inter-module-communication" "01-delivery-service"}}. This time the external call charges a credit card, so the stakes are higher.

That said, the layers are the same ones you've been building.

The board shows the new elements for this exercise: the Place Order [command](https://academy.threedots.tech/knowledge/command), the Bank external service, and the resulting outcomes (events).

{{miroBoard}}

{{message "robert"}}
Customers can now create quotes with locked-in prices. The next step is placing an order from a quote, which means charging the customer.

We prepared a payments client adapter that wraps the bank API and updated `backend/orders/module.go` and `backend/svc.go` to wire everything into the module. The [OpenAPI](https://academy.threedots.tech/knowledge/openapi) spec, [component tests](https://academy.threedots.tech/knowledge/component-test), and [integration tests](https://academy.threedots.tech/knowledge/integration-testing) for the new endpoint are provided.
{{endmessage}}

## The Gateway

This is the first exercise where our backend calls an external API.
In this training and in The Domain Engineer, external services like the bank, tax provider, and file storage are all accessed through a single HTTP gateway.

Instead of storing each service's URL separately, there is one gateway that routes requests to the right service.
The `GATEWAY_ADDR` environment variable holds its address.
When you run `tdl training run`, this variable is filled in automatically with the gateway's address.

Locally, you can start the gateway with `docker compose up`. For tests, set the variable to `http://localhost:8888`. (The Taskfile already has this.)

The `github.com/ThreeDotsLabs/the-domain-engineer` library provides generated OpenAPI clients for the services behind the gateway.
This way you don't need to generate these clients yourself.
Look at `backend/svc.go` to see how the clients are created from `GATEWAY_ADDR`.

The payments client adapter (`backend/orders/adapters/payments/client.go`) wraps the bank client from this library.
You don't write the adapter. Inject it into your service and call it.

## The Order Placement Flow

The endpoint receives `quote_uuid` and `payment_nonce` in the body, with the `Customer-UUID` header for authentication. Here's what the flow looks like:

1. **Retrieve the quote with its menu items.** Fetch the quote and the menu items it references. Use a JOIN between `quote_items` and `restaurant_menu_items` to avoid N+1 queries.

2. **Verify the customer owns the quote.** Compare the quote's customer UUID with the one from the header. If they don't match, return HTTP 403 with error slug `invalid-customer`.

3. **Check the quote hasn't expired.** Quotes have an expiration time. If the quote has passed it, return a `quote-expired` error. The frontend can request a new quote and [retry](https://academy.threedots.tech/knowledge/retry).

4. **Check that no quoted menu items are archived.** Menu items can be archived after the quote was created. **You already have `ensureQuoteItemsAreNotArchived` from quote creation. Call it again here.** If any item is archived, return HTTP 410 Gone with error slug `archived-menu-position` and entity details for each archived item.

5. **Capture payment.** Call the payments service with the nonce, total amount, and the restaurant's UUID as merchant ID. **This call must happen outside any database transaction.** If the payment capture runs inside a transaction and the save rolls back, the payment was still captured by the bank.

6. **Persist the order.** Create an `Order` from the quote (copy financial fields, set `OrderedAt` to now, generate a fresh `OrderUUID`). Save it inside a transaction using `UpdateInTx`.

That's the high-level idea. It's more steps than courier registration, but each one is a few lines.

The **payment nonce** is a one-time token from the frontend's preauthorization (which reserves funds on the customer's card). `CapturePayment` finalizes the charge using this nonce.

The nonce makes the capture [idempotent](https://academy.threedots.tech/knowledge/idempotency). Calling it twice with the same nonce won't double-charge.

{{tip}}
What if payment capture succeeds but saving the order fails? The customer's money left their account, but there's no order in the database. Reconciliation processes can catch this, but they're manual, fragile, and don't scale.

The better approach is the **[Outbox Pattern](https://academy.threedots.tech/knowledge/outbox)**: instead of calling the payment service directly, you emit an event within the same database transaction that saves the order. A separate handler picks up the event and triggers the payment capture. This way, the event and the order data are saved atomically. If the transaction rolls back, the event is never emitted, and no payment is captured.

This is out of scope for this training. If you'd like to learn more, check out our [Go Event-Driven training](https://threedots.tech/event-driven/).
{{endtip}}

## Expected Schema

The `orders.orders` table stores the order. Your migration number is `0006`.

```sql
CREATE TABLE orders.orders
(
    order_uuid              uuid           NOT NULL,
    quote_uuid              uuid           NOT NULL,
    customer_uuid           uuid           NOT NULL,
    restaurant_uuid         uuid           NOT NULL,
    courier_uuid            uuid,
    delivery_address        json           NOT NULL,
    ordered_at              TIMESTAMPTZ    NOT NULL,
    restaurant_confirmed_at TIMESTAMPTZ,
    courier_accepted_at     TIMESTAMPTZ,
    restaurant_prepared_at  TIMESTAMPTZ,
    picked_up_at            TIMESTAMPTZ,
    delivered_at            TIMESTAMPTZ,
    items_subtotal_gross    DECIMAL(10, 2) NOT NULL,
    service_fee_gross       DECIMAL(10, 2) NOT NULL,
    delivery_fee_gross      DECIMAL(10, 2) NOT NULL,
    total_amount_gross      DECIMAL(10, 2) NOT NULL,
    total_tax               DECIMAL(10, 2) NOT NULL,
    currency                varchar(3)     NOT NULL,
    PRIMARY KEY (order_uuid),
    FOREIGN KEY (quote_uuid) REFERENCES orders.quotes (quote_uuid),
    FOREIGN KEY (customer_uuid) REFERENCES orders.customers (customer_uuid),
    FOREIGN KEY (restaurant_uuid) REFERENCES orders.restaurants (restaurant_uuid),
    FOREIGN KEY (courier_uuid) REFERENCES orders.couriers (courier_uuid)
);

CREATE INDEX idx_orders_delivery_city ON orders.orders ((delivery_address->>'city'));
```

Notice that `courier_uuid` is nullable. A courier hasn't been assigned at order placement time.

The lifecycle timestamps (`restaurant_confirmed_at`, `courier_accepted_at`, etc.) are all nullable. They start as null and fill in as the order progresses through later exercises.

Financial fields use `DECIMAL(10, 2)`, same as the quote. The `delivery_address` is stored as JSON, reusing the `shared.Address` type.

## What's Provided

- **Payments client adapter** (`backend/orders/adapters/payments/client.go`): wraps the bank API's `CapturePayment` endpoint.
- **Updated wiring**: `backend/orders/module.go` creates the payments client and passes it to `NewService`. `backend/svc.go` creates the gateway API clients.
- **OpenAPI spec**: already includes the `POST /orders/customer/place-order` endpoint with `PlaceOrder` request and `CustomerOrder` response schemas.
- **Tests**: component tests and integration tests for the new flow are provided.

The `CustomerOrder` response requires a `restaurant_name` field that doesn't come from the order. Your handler needs to fetch it from the restaurant repository.

## Exercise

Exercise path: ./project

**Implement the order placement flow.** `POST /orders/customer/place-order` receives a `PlaceOrder` request with `quote_uuid` and `payment_nonce`, authenticated by the `Customer-UUID` header.

Your endpoint should handle these behaviors:

1. **Happy path**: valid quote and successful payment returns HTTP 201 with the order. Financial fields (items subtotal, service fee, delivery fee, total gross, total tax) should match the original quote.
2. **Customer mismatch**: `Customer-UUID` header doesn't match the quote's customer. Return HTTP 403 with error slug `invalid-customer`.
3. **Expired quote**: quote has passed its expiration time. Return an error with slug `quote-expired`.
4. **Archived menu items**: any menu item archived since the quote was created. Return HTTP 410 with error slug `archived-menu-position` and entity details for each archived item.
5. **Payment capture**: the restaurant's bank account balance should be positive after a successful order.
6. **Order persistence**: the order should exist in the database with correct customer and restaurant associations.
7. Wrap the payments adapter behind an interface and wire it into the service. Same pattern as your repository interfaces.
8. Add column overrides in `backend/orders/adapters/db/sqlc.yaml` for the orders table columns that use dedicated types. Most follow the same approach as quotes and couriers, but `courier_uuid` is nullable — it needs different handling in the sqlc config so the generated type can represent NULL.
9. Run `task gen` (or `go generate ./...`) after writing your migration, queries, and updating `sqlc.yaml`.

{{hints}}

{{hint 1}}
The `courier_uuid` column is nullable, so its sqlc override needs `pointer: true`. Without it, sqlc generates a non-pointer type that can't represent NULL:

```yaml
- column: "orders.orders.courier_uuid"
  go_type:
    import: "eats/backend/orders/app"
    type: "CourierUUID"
    pointer: true
```

The other dedicated type columns (`order_uuid`, `quote_uuid`, etc.) are NOT NULL and follow the same override pattern you've used before.
{{endhint}}

{{hint 2}}

If you're unsure whether your fetch query matches what we expect, here's the structure. Menu items go through `quote_items` as the linking table, so a single quote can have multiple items. That's why we need a `:many` query here. To avoid an N+1, JOIN them in a single query:

```sql
-- name: GetMenuItemsForQuote :many
SELECT
    restaurant_menu_items.*
FROM
    orders.restaurant_menu_items AS restaurant_menu_items
    INNER JOIN orders.quote_items AS quote_items
        ON restaurant_menu_items.restaurant_menu_item_uuid = quote_items.menu_item_uuid
WHERE
    quote_items.quote_uuid = $1
;
```

{{endhint}}

{{endhints}}
