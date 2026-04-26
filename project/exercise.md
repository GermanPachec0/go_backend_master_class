# Quotes Repository

{{message "robert"}}

When a customer picks items from a restaurant's menu, the system creates a **quote**: a snapshot of the price that locks in the subtotal, fees, tax, and delivery cost.
The customer can then place an order from this quote before it expires.

We use this approach so the customer doesn't get surprised by a different price at checkout.
It could happen that a restaurant updates the prices or the delivery fee changes while the customer is browsing the menu.
The quote guarantees the customer pays the price they saw.

We've built the [application layer](https://academy.threedots.tech/knowledge/application-service): validation, the `CreateQuote` service method, application types, and [integration tests](https://academy.threedots.tech/knowledge/integration-testing).
The database migration for `orders.quotes` and `orders.quote_items` tables is also in place.
Your task is, again, the repository.

{{endmessage}}

To create a quote, we need to fetch some details from the database (current prices, currency, address),
do some calculations, and store the quote in the database.

We could do it all in the repository method.
But the logic for turning raw data into a `Quote` belongs in the app layer, not the repository.
Oh, and we want to keep the entire operation in one transaction.
How to best model this?

We'll use the `updateFn` callback pattern.

## The `updateFn` Callback

Take a look at the `OrderRepository` interface:

{{codeFile "backend/orders/app/orders.go"}}

```go
type OrderRepository interface {
    CreateQuote(
        ctx context.Context,
        restaurantUUID RestaurantUUID,
        menuItems CreateQuoteItems,
        updateFn func(
            ctx context.Context,
            menuItems map[RestaurantMenuItemUUID]MenuItem,
            restaurantCurrency shared.Currency,
            restaurantAddress shared.Address,
        ) (Quote, []QuoteMenuItem, error),
    ) (Quote, error)
}
```

`CreateQuote` doesn't receive a ready-made `Quote` struct to persist. Instead, it receives an `updateFn` callback.

The repository fetches data from the database (menu items and restaurant details), passes it to the callback, and the callback returns the quote to persist.

**The callback keeps business logic in the app layer while the repository handles persistence atomically.**

The callback is already written in `Service.CreateQuote`.
It validates that menu items aren't archived, verifies the delivery address is in the restaurant's zone, and calculates pricing.

It returns a `Quote` and `[]QuoteMenuItem` for the repository to persist.

Your job is to call it with the right data and persist what it returns.

## Batch Inserts with `:copyfrom`

A quote can have many items. You could insert them one by one in a loop, but sqlc supports a better way.
**The [`:copyfrom`](https://docs.sqlc.dev/en/latest/howto/insert.html#using-copyfrom) annotation generates a batch insert using PostgreSQL's [COPY protocol](https://www.postgresql.org/docs/current/sql-copy.html).**
You can pass all items in a single call.

The SQL looks like a regular INSERT:

```sql
-- name: AddQuoteItems :copyfrom
INSERT INTO orders.quote_items (
    quote_item_uuid,
    quote_uuid,
    menu_item_uuid,
    gross_price,
    quantity
)
VALUES
    ($1, $2, $3, $4, $5);
```

The `:copyfrom` annotation makes sqlc generate a function that takes a slice:

```go
func (q *Queries) AddQuoteItems(ctx context.Context, arg []AddQuoteItemsParams) (int64, error)
```

It'll insert all the items at once. The `int64` return value is the number of inserted rows.

For a handful of items, individual inserts would work fine. But `:copyfrom` scales better, and the syntax is simpler.

{{tip}}

For a deeper look at the callback pattern and why we use one repository per [Aggregate](https://academy.threedots.tech/knowledge/aggregate) (not per table), see [The Repository Pattern in Go](https://threedots.tech/post/repository-pattern-in-go/) and [Database Transactions in Go](https://threedots.tech/post/database-transactions-in-go/).

{{endtip}}

## Exercise

Exercise path: ./project

Start with a few supporting files:

1. In `backend/orders/adapters/db/queries/orders.sql` add two queries:
    * `AddQuote :exec` that inserts into `orders.quotes`.
    * `AddQuoteItems :copyfrom` that inserts into `orders.quote_items`.
2. In `backend/orders/adapters/db/queries/restaurants.sql` add a `GetMenuItemsByUUIDs :many` query that fetches menu items filtered by restaurant UUID and a list of menu item UUIDs.
Use the `ANY ($2::UUID[])` syntax you learned in the restaurants repository.
3. In `backend/orders/adapters/db/sqlc.yaml` add type overrides for `orders.quotes` and `orders.quote_items` columns, following the same pattern you used for restaurants.

Run `task gen` or `go generate ./...` after adding the queries.

Then, **implement the `OrderRepository` interface**:

4. Create a new file: `backend/orders/adapters/db/orders_repo.go`.
5. Inside, add an `OrdersRepo` struct with a constructor (just like in the other repositories).
6. Add the `CreateQuote` method, (implement the `app.OrderRepository` interface).
Here's the flow inside the repository's `CreateQuote`:

    - Receive `restaurantUUID`, `menuItems` (what the customer wants), and `updateFn`
    - Inside `UpdateInTx`, fetch the menu items from the database using their UUIDs
    - Fetch the restaurant for its currency and address
    - Call `updateFn` with the fetched data. It returns a `Quote` and `[]QuoteMenuItem`
    - Persist the quote with an `AddQuote` query
    - Persist all quote items with `AddQuoteItems`
    - Return the created quote

**Remember: pass `tx` to `dbmodels.New()`, not `r.db`.**
All queries must run inside one transaction.
The `tx` comes from the `UpdateInTx` callback.

Most of the work is in `orders_repo.go`. The SQL queries and config are a few lines each.

The platform runs integration tests in `backend/orders/adapters/db/orders_repo_test.go`.

{{hints}}

{{hint 1}}

Here's the structure of the `CreateQuote` method:

```go
func (r *OrdersRepo) CreateQuote(
    ctx context.Context,
    restaurantID app.RestaurantUUID,
    menuItems app.CreateQuoteItems,
    updateFn func(
        ctx context.Context,
        menuItems map[app.RestaurantMenuItemUUID]app.MenuItem,
        restaurantCurrency shared.Currency,
        restaurantAddress shared.Address,
    ) (app.Quote, []app.QuoteMenuItem, error),
) (app.Quote, error) {
    var quote app.Quote

    err := common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
        queries := dbmodels.New(tx)

        // TODO

        return nil
    })

    return quote, err
}
```

You'll need a helper to convert `[]app.QuoteMenuItem` to `[]dbmodels.AddQuoteItemsParams`, and another to convert the DB menu item rows to `map[app.RestaurantMenuItemUUID]app.MenuItem`.

{{endhint}}

{{hint 2}}

You can use these SQL queries:

```sql
-- name: AddQuote :exec
INSERT INTO orders.quotes (
	quote_uuid,
	customer_uuid,
	restaurant_uuid,
	delivery_address,
	items_subtotal_gross,
	service_fee_gross,
	delivery_fee_gross,
	total_amount_gross,
	total_tax,
	created_at,
	currency
)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
;

-- name: AddQuoteItems :copyfrom
INSERT INTO orders.quote_items (
	quote_item_uuid,
	quote_uuid,
	menu_item_uuid,
	gross_price,
	quantity
)
VALUES
	($1, $2, $3, $4, $5);
```

{{endhint}}

{{hint 3}}

You need this query to fetch menu items by their UUIDs:

```sql
-- name: GetMenuItemsByUUIDs :many
SELECT
	restaurant_menu_items.*
FROM
	orders.restaurant_menu_items AS restaurant_menu_items
WHERE
	restaurant_uuid = $1 AND
	restaurant_menu_item_uuid = ANY ($2::UUID[])
;
```

{{endhint}}

{{hint 4}}

Here's the sqlc mapping you need:

```yaml
          - column: "orders.quotes.customer_uuid"
            go_type:
              import: "eats/backend/orders/app"
              type: "CustomerUUID"

          - column: "orders.quotes.restaurant_uuid"
            nullable: true
            go_type:
              import: "eats/backend/orders/app"
              type: "RestaurantUUID"

          - column: "orders.quotes.quote_uuid"
            go_type:
              import: "eats/backend/orders/app"
              type: "QuoteUUID"

          - column: "orders.quotes.currency"
            go_type:
              import: "eats/backend/common"
              type: "Currency"

          - column: "orders.quotes.delivery_address"
            go_type:
              import: "eats/backend/common"
              type: "Address"

          - column: "orders.quote_items.quote_uuid"
            go_type:
              import: "eats/backend/orders/app"
              type: "QuoteUUID"

          - column: "orders.quote_items.menu_item_uuid"
            go_type:
              import: "eats/backend/orders/app"
              type: "RestaurantMenuItemUUID"
```

{{endhint}}

{{endhints}}
