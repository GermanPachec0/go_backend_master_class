# Restaurants Repository

{{message "milosz"}}

With customers able to register, let's move to onboarding restaurants.
Each restaurant needs an account created before they can receive orders.

We've already implemented the [application layer](https://academy.threedots.tech/knowledge/application-service) for it.
The `OnboardRestaurant` service method takes care of validation and calls the repository.

Your task is to implement the repository method.
There are [integration tests](https://academy.threedots.tech/knowledge/integration-testing) ready for it that verify the behavior you need to implement.

{{endmessage}}

The customer repository was just a single INSERT.
Restaurant onboarding is more complex: a restaurant comes with menu items, and they need to be saved alongside the restaurant itself.

Most of the code is already there, but you need to fill the gaps in the `UpsertRestaurant` method.

## Upserts

We want to create an *upsert* method: a query that inserts a new restaurant if it doesn't exist, or updates the existing one if it does.
This way, the same method can be used for both creating and updating a restaurant.

The same goes for each menu item.
We need a query that handles both cases: insert if new, update if existing.

**Upserts are [idempotent](https://academy.threedots.tech/knowledge/idempotency). Calling an upsert twice with the same data produces the same result.**
This is important for [retry](https://academy.threedots.tech/knowledge/retry) logic and for making sure your endpoints are safe to call multiple times.

In PostgreSQL, we can use `INSERT ... ON CONFLICT DO UPDATE` as an upsert.

Take a look at the `UpsertRestaurant` query you already have:

{{codeFile "backend/orders/adapters/db/queries/restaurants.sql"}}

```sql
INSERT INTO orders.restaurants (restaurant_uuid, name, description, address, currency)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (restaurant_uuid) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    address = EXCLUDED.address
RETURNING *;
```

The `UPDATE` part runs if there's a conflict on `restaurant_uuid` (another row with this UUID already exists).

The [`EXCLUDED`](https://www.postgresql.org/docs/current/sql-insert.html#SQL-ON-CONFLICT) keyword refers to the row that we tried to insert.
So `EXCLUDED.name` means "the name value from the INSERT that conflicted."

Notice that `currency` is NOT in the `SET` clause.
Once a restaurant is created with a currency, it shouldn't change.

The `UpsertRestaurant` method we created already checks this: after the upsert, if the returned `currency` differs from the requested one, it returns an error.

{{conversation "From a Past Code Review"}}

{{message "robert"}}

Should we add a generic `UpdateRestaurant(fields map[string]interface{})` method? That way, we don't need a new repository method every time something changes.

{{endmessage}}

{{message "milosz" "robert:+1"}}

Specific methods like `UpsertRestaurant` tell you exactly what's happening at the call site. With a generic update, you lose type safety and can't tell from the code which fields are being modified. It's also harder to review and audit. If we need more operations later, we add more methods.

{{endmessage}}

{{endconversation}}

We'll need a similar query for upserting menu items. Something like:

```sql
INSERT INTO orders.restaurant_menu_items (
	restaurant_menu_item_uuid,
	restaurant_uuid,
	name,
	gross_price,
	ordering,
	is_archived
)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (restaurant_menu_item_uuid) DO UPDATE SET ...
```

## sqlc.embed

To verify the data you save, you'll also need a query that fetches menu items.
[`sqlc.embed`](https://docs.sqlc.dev/en/latest/howto/embedding.html) tells sqlc to generate a nested struct for the table's columns instead of flattening them:

```sql
SELECT sqlc.embed(restaurant_menu_items)
FROM orders.restaurant_menu_items AS restaurant_menu_items
WHERE restaurant_uuid = $1 AND is_archived = FALSE
ORDER BY ordering ASC
```

**Without `sqlc.embed`, you'd access fields as `dbItem.RestaurantMenuItemUuid`. With it, fields are nested: `dbItem.OrdersRestaurantMenuItem.RestaurantMenuItemUuid`.**

The nested struct prevents field name collisions when joining multiple tables. We're using it here so the query is ready for future JOINs.

Note: when using embed, the table alias (`AS restaurant_menu_items`) is required.

## Handling Money with Decimal Types

The `restaurant_menu_items` table uses a `DECIMAL(10,2)` column for `gross_price`.
By default, sqlc maps this to `pgtype.Numeric`, which is awkward to work with.
We need to configure sqlc to use [shopspring/decimal](https://github.com/shopspring/decimal) instead.

The override goes in the `overrides` section of `backend/orders/adapters/db/sqlc.yaml`:

```yaml
- db_type: "pg_catalog.numeric"
  go_type:
    import: "github.com/shopspring/decimal"
    type: "Decimal"
```

Once it's in place, all database decimals will be represented by `decimal.Decimal`.

{{tip}}
**Never use `float64` for money!**

Floating point math has precision issues that will cause real bugs in financial calculations:

```go
// This prints 9.999999999999831, not 10
func main() {
	var n float64 = 0
	for i := 0; i < 1000; i++ {
		n += 0.01
	}
	fmt.Println(n)
}

```

The [shopspring/decimal](https://github.com/shopspring/decimal) library provides arbitrary precision decimal arithmetic, making it safe for financial calculations.

Read more: [Why don't you just use float64?](https://github.com/shopspring/decimal?tab=readme-ov-file#why-dont-you-just-use-float64)
{{endtip}}

{{tip}}

You can run the tests locally with `task test-integration` if you want a faster feedback loop. Make sure Docker is running first (`task up`). This is optional. `tdl tr run` handles everything for you.

If you don't use Task, run `docker compose up` and `go test -tags=integration ./...`.

{{endtip}}

## Exercise

Exercise path: ./project

The integration tests in `backend/orders/adapters/db/restaurant_repo_test.go` verify that:

- Restaurants can be created and updated through upserts
- Menu items are upserted correctly (names, prices, ordering)
- Currency can't change after the restaurant is created
- Repeated calls with the same data produce the same result **(idempotency)**

Here's what you need to do:

1. Add the decimal type mapping to `backend/orders/adapters/db/sqlc.yaml`
2. Add the SQL queries to `backend/orders/adapters/db/queries/restaurants.sql`:
    * `GetRestaurantMenu` - using `sqlc.embed` to return all active menu items for the restaurant
    * `UpsertRestaurantMenuItem` - inserting a new menu item or updating an existing one (use `ON CONFLICT DO UPDATE`)
3. Run `task gen` or `go generate ./...` to regenerate the Go code for the new queries
4. **Add upserting menu items to `UpsertRestaurant` in `backend/orders/adapters/db/restaurant_repo.go`**:
    * Loop through `restaurant.MenuItems` and upsert each one

{{tip}}

You may notice that we're not using transactions here, and there's no way to remove menu items yet. We'll add both in the next exercise.

{{endtip}}

{{hints}}

{{hint 1}}

Because menu items use the restaurant UUID as a foreign key, you must upsert menu items after upserting the restaurant itself.
Otherwise, you'd get foreign key violations when trying to insert menu items for a restaurant that doesn't exist yet.

{{endhint}}

{{hint 2}}

You need these queries in `restaurants.sql`:

```sql
-- name: GetRestaurantMenu :many
SELECT
	sqlc.embed(restaurant_menu_items)
FROM
	orders.restaurant_menu_items AS restaurant_menu_items
WHERE
	restaurant_uuid = $1 AND
	is_archived = FALSE
ORDER BY
	ordering ASC
;

-- name: UpsertRestaurantMenuItem :exec
INSERT INTO orders.restaurant_menu_items (
	restaurant_menu_item_uuid,
	restaurant_uuid,
	name,
	gross_price,
	ordering,
	is_archived
)
VALUES
	($1, $2, $3, $4, $5, $6)
ON CONFLICT (restaurant_menu_item_uuid) DO UPDATE SET
	restaurant_uuid = EXCLUDED.restaurant_uuid,
	name = EXCLUDED.name,
	gross_price = EXCLUDED.gross_price,
	ordering = EXCLUDED.ordering,
	is_archived = EXCLUDED.is_archived
;
```

{{endhint}}

{{hint 3}}

The upsert loop is straightforward. After the restaurant upsert (which is already in place), loop through the incoming menu items:

```go
for _, item := range restaurant.MenuItems {
    err = queries.UpsertRestaurantMenuItem(ctx, dbmodels.UpsertRestaurantMenuItemParams{
        RestaurantMenuItemUuid: item.MenuItemUUID,
        RestaurantUuid:         restaurantUUID,
        Name:                   item.Name,
        GrossPrice:             item.GrossPrice,
        Ordering:               item.Ordering,
        IsArchived:             false,
    })
    if err != nil {
        return fmt.Errorf("upsert restaurant menu position failed for menu position %s: %w", item.MenuItemUUID, err)
    }
}
```

{{endhint}}

{{endhints}}
