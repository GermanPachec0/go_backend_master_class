# Menu Item Archiving

{{message "robert"}}

Restaurants can now be created and updated, along with their menu items.
But what happens when a restaurant removes a menu item from their offering?

We don't want to delete the row. Other parts of the system (like past orders) may still reference it.
Instead, we archive the item: set `is_archived = TRUE` so it no longer shows up in the active menu.

{{endmessage}}

The upsert handles adding and updating menu items. But it doesn't handle *removal*.
If a restaurant sends an updated menu without an item that was there before, we need to detect that and archive it.

The full upsert flow now looks like this:

1. Fetch current menu items from the database
2. Upsert the restaurant record
3. Upsert each menu item
4. Compare current items against the incoming list to find which ones were removed
5. Archive the removed items

All of these operations need to happen atomically.
If one of them fails, the database would end up in an inconsistent state.
That's why we wrap everything in a transaction.

A database transaction groups multiple queries into a single unit of work.
Either all of them succeed (commit), or none of them take effect (rollback).
Without a transaction, a failure halfway through would leave the database with some changes applied and others not.

## Transactions and UpdateInTx

We prepared a helper for this: `common.UpdateInTx`.

```go
return common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
    queries := dbmodels.New(tx) // Use queries with the transaction
    // ... all operations use the same transaction
    return nil
})
```

It takes an anonymous function as an argument. Whatever happens inside that function and uses the `tx` is part of one transaction.
If the function returns an error, the transaction rolls back automatically. If it returns nil, the transaction commits.

It starts a transaction with [Repeatable Read](https://www.postgresql.org/docs/current/transaction-iso.html#XACT-REPEATABLE-READ) isolation, which prevents phantom reads.
If two concurrent transactions conflict, PostgreSQL raises a serialization error, and `UpdateInTx` [retries](https://academy.threedots.tech/knowledge/retry) automatically with backoff.

The key is passing `tx` to `dbmodels.New()`, instead of `r.db`.
This makes all queries use the same transaction.

{{tip}}
**All code inside the `UpdateInTx` closure re-executes on every retry.** When PostgreSQL detects a serialization conflict, `UpdateInTx` rolls back and runs the entire function again. This is fine for database queries, but if you put HTTP calls, [gRPC](https://academy.threedots.tech/knowledge/grpc) calls, or any external service calls inside the closure, those get repeated too.

Move external calls outside the closure. Compute everything you need first, then pass the results into the transaction. If you need strong consistency between the external call and the database write, use explicit locking (e.g., `SELECT FOR UPDATE`) instead of relying on retries.

We'll see this pattern in action when we implement handlers in later modules.
{{endtip}}

## The ANY Operator

To archive multiple items in a single query, we can use the `ANY` operator with a UUID array:

```sql
UPDATE orders.restaurant_menu_items
SET is_archived = TRUE
WHERE restaurant_menu_item_uuid = ANY ($1::UUID[])
```

It's similar to `WHERE id IN (1, 2, 3)`, but `IN` requires a fixed list of values. **`ANY` works with a single array parameter, which is what sqlc needs.**

The `$1::UUID[]` syntax casts the parameter to a UUID array type, and sqlc maps it to `[]common.UUID` in Go.

## Exercise

Exercise path: ./project

The [integration tests](https://academy.threedots.tech/knowledge/integration-testing) verify that removed menu items are **archived, not deleted**.
An item that was in the previous menu but is missing from the new one should no longer appear in `GetRestaurantMenu` results.

Here's what you need to do:

1. Add the `ArchiveMenuItems` query to `backend/orders/adapters/db/queries/restaurants.sql`:
    * Use `ANY` with a UUID array to archive multiple items in a single query
2. Run `task gen` or `go generate ./...` to regenerate the Go code
3. **Wrap `UpsertRestaurant` in `common.UpdateInTx`** in `backend/orders/adapters/db/restaurant_repo.go`:
    * Pass `tx` to `dbmodels.New()` instead of `r.db`
    * Before upserting, fetch current menu items with `GetRestaurantMenu`
    * After upserting, compare current UUIDs against incoming UUIDs to find removed items
    * Archive any removed items
4. **Wrap `RegisterCustomer` in `common.UpdateInTx`** in `backend/orders/adapters/db/customer_repo.go`:
    * While a transaction is not required for a single INSERT, it'll come in handy when we extend customer registration to include more operations.

{{hints}}

{{hint 1}}

The full flow inside `UpsertRestaurant` has five steps, all within a single `UpdateInTx` call:

1. `GetRestaurantMenu` to fetch what's currently in the database
2. `UpsertRestaurant` to insert or update the restaurant record (already implemented)
3. Loop through incoming `restaurant.MenuItems`, calling `UpsertRestaurantMenuItem` for each (already implemented)
4. Compare current menu item UUIDs against the incoming list to find removed items
5. Call `ArchiveMenuItems` for any existing items not in the new list

{{endhint}}

{{hint 2}}

You need this query in `restaurants.sql`:

```sql
-- name: ArchiveMenuItems :exec
UPDATE
	orders.restaurant_menu_items
SET
	is_archived = TRUE
WHERE
	restaurant_menu_item_uuid = ANY ($1::UUID[])
;
```

{{endhint}}

{{hint 3}}

The archive logic compares two lists. Collect UUIDs of current items (from `GetRestaurantMenu`), then loop through them.
For each current UUID, check if it exists in the incoming `restaurant.MenuItems`.
If not found, add it to a `toArchive` slice:

```go
menuItemsToArchive := make([]common.UUID, 0)
for _, currentUUID := range currentMenuItemsUUIDs {
    found := false
    for _, newItem := range restaurant.MenuItems {
        if currentUUID == newItem.MenuItemUUID {
            found = true
            break
        }
    }
    if !found {
        menuItemsToArchive = append(menuItemsToArchive, currentUUID.UUID)
    }
}
```

Then, if `len(menuItemsToArchive) > 0`, call `ArchiveMenuItems` with that slice.

{{endhint}}

{{endhints}}
