# Restaurant Order Management

The customer paid and the order is saved. But from the restaurant's perspective, nothing has happened yet. Time to build the other side: accepting orders and marking them ready for pickup.

Two new commands on the board: Accept Order and Mark As Prepared.

{{miroBoard}}

## The Two Endpoints

Both endpoints follow the same structure:

- `POST /orders/restaurant/accept-order` sets `restaurant_confirmed_at` when the restaurant accepts the order. Authenticated by the `Restaurant-UUID` header, the request body contains `AcceptOrder` with `order_uuid`. Returns **202**.
- `POST /orders/restaurant/mark-order-ready-for-pickup` sets `restaurant_prepared_at` when the food is ready. Same authentication, same shape. Request body: `MarkOrderReady` with `order_uuid`. Returns **202**.

Both return 202 (not 201) because these are status transitions, not resource creation. The structure is the same for both.

The service methods use the `updateFn` callback pattern from {{exerciseLink "Quotes Repository" "08-advanced-repositories" "05-quotes-repository"}}. Same read-modify-write approach, now applied to order status transitions.

## COALESCE for Partial Updates

The orders table has many timestamp columns, but each operation only sets one. You need an UPDATE query that changes a single column without touching the rest.

**COALESCE combined with `sqlc.narg` solves this: pass NULL for columns you don't want to change, and the existing value is preserved.**

```sql
-- name: UpdateOrder :exec
UPDATE
    orders.orders
SET
    courier_uuid = COALESCE(sqlc.narg(courier_uuid), courier_uuid),
    ordered_at = COALESCE(sqlc.narg(ordered_at), ordered_at),
    restaurant_confirmed_at = COALESCE(sqlc.narg(restaurant_confirmed_at), restaurant_confirmed_at),
    courier_accepted_at = COALESCE(sqlc.narg(courier_accepted_at), courier_accepted_at),
    restaurant_prepared_at = COALESCE(sqlc.narg(restaurant_prepared_at), restaurant_prepared_at),
    picked_up_at = COALESCE(sqlc.narg(picked_up_at), picked_up_at),
    delivered_at = COALESCE(sqlc.narg(delivered_at), delivered_at)
WHERE
    order_uuid = $1;
```

What's counter-intuitive is that **NULL here means "no change," not "clear the field."** `sqlc.narg` generates a pointer parameter. When you pass `nil`, sqlc sends NULL, and COALESCE falls back to the current column value.

You already know `sqlc.narg` from {{exerciseLink "Ordering and Filtering" "09-read-models" "02-ordering-filtering"}}, where it was used in WHERE clauses. Here it's used in SET clauses. Same tool, new application.

This single query serves all future status transitions too (courier delivery, pickup, etc.), not only this exercise. It's good enough for our purposes.

## Idempotent Operations

Both operations should be **[idempotent](https://academy.threedots.tech/knowledge/idempotency)**. If a restaurant accepts the same order twice, `restaurant_confirmed_at` should not change. Network [retries](https://academy.threedots.tech/knowledge/retry) and user double-clicks mean duplicate requests happen in production.

**If the timestamp is already set, log a warning and return the order unchanged.** No error, no side effect.

## Authorization

Both endpoints verify that the requesting restaurant owns the order. Compare the order's restaurant UUID against the `Restaurant-UUID` header. If they don't match, return **403** with error slug `"invalid-restaurant"`.

This is an authorization problem (wrong actor), not a validation problem (bad input). That's why 403, not 400. HTTP 401 means "not authenticated" and fits when credentials are missing entirely. HTTP 403 means "authenticated but not allowed": the restaurant proved who it is, but it's trying to access someone else's order. Use `common.NewForbiddenError`, which follows the same pattern as `NewUnauthorizedError` from module 07.

For now, that's all you need. No rejection, no cancellation.

## Exercise

Exercise path: ./project

**Implement restaurant order management.** Two endpoints let a restaurant accept an order and mark it ready for pickup.

1. Add a `GetOrder` SQL query in `backend/orders/adapters/db/queries/orders.sql` that fetches an order by `order_uuid`.

2. Add the `UpdateOrder` SQL query (shown above) using COALESCE with `sqlc.narg` for partial updates. Run `task gen` to regenerate sqlc code. (If you don't use Task, run `go generate ./...` instead.)

3. **Add `OrderByID` and `UpdateOrder` methods to the `OrderRepository` interface** in the [application layer](https://academy.threedots.tech/knowledge/application-service). `OrderByID` should return a 404 error with slug `"order_not_found"` when the order doesn't exist.

4. Implement `OrderByID` in the repository. Handle `pgx.ErrNoRows` by returning `common.NewNotFoundError("order_not_found", ...)`.

5. **Implement `UpdateOrder` in the repository using the `updateFn` callback pattern with `UpdateInTx`**: read the order inside the transaction, call `updateFn`, persist the result.

6. Add service methods for accepting the order and marking it ready for pickup. Inside each `updateFn` callback:
   - Verify the restaurant owns the order (403 if not)
   - Check if the timestamp is already set. If so, log a warning and return the order unchanged (idempotency)
   - Set the appropriate timestamp

7. Add HTTP handler methods that delegate to the service and return 202.

Your endpoints should handle these behaviors:
- `POST /orders/restaurant/accept-order` returns 202 on success
- `POST /orders/restaurant/mark-order-ready-for-pickup` returns 202 on success
- Wrong restaurant gets 403 for either endpoint
- Accepting an already-accepted order preserves the original `restaurant_confirmed_at` timestamp

{{tip}}
When mapping between database types and application types, consider using positional (unnamed) struct fields instead of named fields:

```go
return app.Order{
    dbOrder.OrderUuid,
    dbOrder.QuoteUuid,
    dbOrder.CustomerUuid,
    dbOrder.RestaurantUuid,
    dbOrder.CourierUuid,
    dbOrder.DeliveryAddress,
    dbOrder.OrderedAt,
    // TODO: remaining timestamp and amount fields
}
```

If someone adds a new field to the `Order` struct later, this code won't compile until it's updated. With named fields, the new field silently gets a zero value. It's more verbose, but it catches missing fields at compile time.
{{endtip}}

{{hints}}

{{hint 1}}
The `updateFn` callback runs inside the transaction, after the order is read from the database. Both the authorization check and the idempotency check go inside this callback:

```go
func(ctx context.Context, order Order) (Order, error) {
    if err := checkRestaurantMatch(order.RestaurantUUID, restaurantUUID); err != nil {
        return Order{}, err
    }

    if order.RestaurantConfirmedAt != nil {
        // Already confirmed -- idempotent, just return unchanged
        return order, nil
    }
    order.RestaurantConfirmedAt = common.ToPtr(time.Now())

    return order, nil
}
```

The mark-ready-for-pickup method follows the same shape but checks and sets `RestaurantPreparedAt`.
{{endhint}}

{{hint 2}}
The `UpdateOrder` repository method follows the same structure as the quotes repository from module 08:

```go
func (r *OrdersRepo) UpdateOrder(
    ctx context.Context,
    orderUUID app.OrderUUID,
    updateFn func(ctx context.Context, order app.Order) (app.Order, error),
) error {
    return common.UpdateInTx(ctx, r.db, func(ctx context.Context, tx pgx.Tx) error {
        queries := dbmodels.New(tx)

        dbOrder, err := queries.GetOrder(ctx, orderUUID)
        if err != nil {
            return fmt.Errorf("failed to get order: %w", err)
        }

        updatedOrder, err := updateFn(ctx, dbOrderToAppOrder(dbOrder))
        if err != nil {
            return fmt.Errorf("failed to update order: %w", err)
        }

        return queries.UpdateOrder(ctx, dbmodels.UpdateOrderParams{
            orderUUID,
            updatedOrder.CourierUUID,
            &updatedOrder.OrderedAt,
            updatedOrder.RestaurantConfirmedAt,
            updatedOrder.CourierAcceptedAt,
            updatedOrder.RestaurantPreparedAt,
            updatedOrder.PickedUpAt,
            updatedOrder.DeliveredAt,
        })
    })
}
```

Pass each timestamp field as a pointer. Fields you set will have values; fields you didn't touch stay `nil`, and COALESCE preserves the existing database value.
{{endhint}}

{{endhints}}
