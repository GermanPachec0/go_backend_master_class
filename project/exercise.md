# Simple Read Model

{{message "milosz"}}

Before the users can create a quote and place an order, they need to select what to buy.

Instead of the usual "select restaurant to see the menu" flow, we want to show a list of all available menu items across all restaurants.
Each item needs to include the restaurant name, so users can filter by restaurant if they want.

{{endmessage}}

If we wanted to re-use the existing application types, showing a list of menu items for all restaurants would be tricky.

We'd need to list all restaurants, then for each fire another query to fetch menu items.
With 100 restaurants averaging 20 menu items each, that's 101 SQL queries and 2,000+ objects loaded into memory, all to show a simple list.

This is the **N+1 problem**, one of the most common performance issues in web applications.

The database can do this in a single query with a JOIN.
It's a straightforward SQL query, too.
**The problem starts if we try to route it through our existing application types** that weren't designed for this use case.

## Read Models

Once you start using application types, it feels natural to want to keep one model for each [entity](https://academy.threedots.tech/knowledge/entity).
If you have a `restaurants` table, there is a `Restaurant` application type, and any duplication feels wrong.

But with software design, you need to know when it makes sense to break the rules.
This is one of those cases.

**For read-only queries, we can skip the application layers entirely.** A [read model](https://academy.threedots.tech/knowledge/read-model) is a data structure that:

- comes from data joined directly in SQL,
- returns exactly the structure the client needs,
- doesn't go through application entities or services.

We run a single query instead of 101 and return HTTP response types directly from the database layer.

We already set up the infrastructure for this:

{{codeFile "backend/orders/adapters/db/read_model.go"}}

```go
type ReadModel struct {
	db *pgxpool.Pool
}

func NewReadModel(db *pgxpool.Pool) *ReadModel {
	if db == nil {
		panic("db connection pool cannot be nil")
	}
	return &ReadModel{db: db}
}

func (r ReadModel) ListMenuItemsWithRestaurant(ctx context.Context) ([]http.MenuItemWithRestaurant, error) {
	return nil, errors.New("not implemented")
}
```

You can think of this as a kind of [repository](https://academy.threedots.tech/knowledge/repository-pattern), although much simpler.
It runs read-only queries and has no logic that repositories sometimes need to have.

The `ListMenuItemsWithRestaurant` method will use sqlc to run a single JOIN query and map the rows to `http.MenuItemWithRestaurant` types.
It won't touch application types at all.

## Why HTTP Types Are Fine Here

In the {{exerciseLink "Application Layer Types" "06-application-layer" "01-app-types"}} exercise, we saw that returning HTTP types from a repository couples the database layer to the API.
That's true for writes, where data flows through the [application service](https://academy.threedots.tech/knowledge/application-service) and types, with some logic in between.

Read models are a different case. **A read model exists to serve a specific consumer.**
If you add a `rating` field to the API response, you update the SQL query too.
They always change together, so that coupling is fine.

If we created intermediate DTOs (Data Transfer Objects) between the query and the HTTP response, we'd get zero benefit and just unnecessary boilerplate.
For read-only queries, returning HTTP types directly is all you need.

This separation of reads and writes is one the most important ideas in software design.
**Writes enforce business rules and must be consistent.
Reads display data efficiently and match what the client needs.**

If you'd like to read more about separating read and write concerns, see [How to use basic CQRS in Go](https://threedots.tech/post/basic-cqrs-in-go/)
and [Killing the legacy and other CQRS stories](https://www.youtube.com/watch?v=GdLu7FQBrdk).

{{tip}}

**Read models can join data from different modules.**
In this exercise, the query joins tables within the `orders` schema.
In larger systems, you might need to join data from tables owned by different modules.

For a small team working on a single codebase, occasional joining across schemas for read models is fine.

You need to be careful to avoid too much coupling between modules this way.
Your modules are no longer fully independent once you do it.
Still, it can be a pragmatic choice if the alternative is running a complex GraphQL server.

If modules are owned by different teams, one team changing their schema can break another team's read model.
In that scenario, consider event-driven read models with denormalized storage.
We cover this pattern in our [Go Event-Driven training](https://threedots.tech/event-driven/).

{{endtip}}

## Exercise

Exercise path: ./project

1. **Create `backend/orders/adapters/db/queries/read_models.sql`** and add a new query there.
    * `-- name: ListMenuItemsWithRestaurant :many`
    * Join `restaurant_menu_items` and `restaurants` on `restaurant_uuid`.
    * Filters out archived items (`WHERE mi.is_archived = false`)
    * Order results by restaurant name, then by item ordering within each restaurant.
2. **Run `task gen`** to regenerate the Go code.
3. **Implement `ListMenuItemsWithRestaurant`** in the existing `ReadModel` at `backend/orders/adapters/db/read_model.go`.
    * Run the query your created and map rows to `[]http.MenuItemWithRestaurant`.

The platform will verify that a restaurant can be onboarded with menu items and that the read model endpoint returns the data correctly.

{{hints}}

{{hint 1}}

```sql
-- name: ListMenuItemsWithRestaurant :many
SELECT
    mi.restaurant_menu_item_uuid AS menu_item_uuid,
    mi.name AS menu_item_name,
    mi.gross_price,
    r.currency,
    r.restaurant_uuid,
    r.name AS restaurant_name
FROM orders.restaurant_menu_items mi
JOIN orders.restaurants r ON mi.restaurant_uuid = r.restaurant_uuid
WHERE mi.is_archived = false
ORDER BY r.name, mi.ordering;
```

{{endhint}}

{{hint 2}}

After running `task gen`, use the generated query from `dbmodels`. An example implementation:

```go
func (r ReadModel) ListMenuItemsWithRestaurant(ctx context.Context) ([]http.MenuItemWithRestaurant, error) {
	queries := dbmodels.New(r.db)

	rows, err := queries.ListMenuItemsWithRestaurant(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]http.MenuItemWithRestaurant, 0, len(rows))
	for _, row := range rows {
		items = append(items, http.MenuItemWithRestaurant{
			MenuItemUuid:   row.MenuItemUuid,
			MenuItemName:   row.MenuItemName,
			GrossPrice:     row.GrossPrice,
			Currency:       row.Currency,
			RestaurantUuid: row.RestaurantUuid,
			RestaurantName: row.RestaurantName,
		})
	}

	return items, nil
}
```

{{endhint}}

{{endhints}}
