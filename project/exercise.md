# Read Models and Listing Endpoints

The order lifecycle works end to end. You can place an order, the restaurant accepts and prepares it, a courier picks it up and delivers it. But all of this only works because your tests pass UUIDs directly.

A real restaurant opens a dashboard and sees incoming orders. A courier opens their app to find available deliveries in their city. Nobody types in a UUID. That's where listing endpoints come in.

We covered the [read model](https://academy.threedots.tech/knowledge/read-model) pattern in the {{exerciseLink "Simple Read Model" "09-read-models" "01-simple-read-model"}} exercise. Read models return HTTP types directly, bypassing the [application layer](https://academy.threedots.tech/knowledge/application-service). What's new here is that we're building **four read models from the same underlying orders table, each shaped for a different consumer**.

The Order view now appears on the board.
Notice how it feeds into the "Accept Order" [command](https://academy.threedots.tech/knowledge/command).
On an [Event Storming](https://academy.threedots.tech/knowledge/event-storming) board, views can be inputs to commands.
The restaurant sees a list of incoming orders (the view), picks one, and accepts it (the command).

{{miroBoard}}

In practice, each actor needs a different view of the same order data. This is the opposite of a CRUD approach where one model serves all consumers, and it fits real use cases of complex applications much better:

| | Customer | Restaurant | Courier |
|---|---|---|---|
| Pricing fields | `delivery_fee_gross`, `service_fee_gross`, `total_amount_gross`, `total_tax` | `items_subtotal_gross` only | `items_subtotal_gross` only |
| Restaurant info | `restaurant_name` (via JOIN) | not needed (they know their own name) | `restaurant_name` (via JOIN) |
| Customer info | not needed (it's their own data) | `customer_uuid` | `customer_uuid` |
| Delivery address | included | not included | included |
| Timestamps | all lifecycle timestamps | all lifecycle timestamps | all lifecycle timestamps |
| Filtering | by `customer_uuid` | by `restaurant_uuid` | by `courier_uuid` (assigned) or by city + availability (available) |

**Each actor gets a dedicated [OpenAPI](https://academy.threedots.tech/knowledge/openapi) schema (`CustomerOrder`, `RestaurantOrder`, `CourierOrder`) because a single Order type with everything nullable would hide the API contract.** The OpenAPI spec with all four endpoints and schemas is already provided.

For most applications, querying from a single database like this is good enough.

## Implementation Pattern

For each endpoint, the implementation follows the same three-layer pattern from the {{exerciseLink "Simple Read Model" "09-read-models" "01-simple-read-model"}} exercise:

1. SQL query in `backend/orders/adapters/db/queries/orders.sql` with appropriate JOINs and WHERE clauses
2. ReadModel method in `backend/orders/adapters/db/read_model.go` that maps database rows to HTTP response types
3. Handler method in `backend/orders/api/http/handler.go` that delegates to the read model

The handler methods are thin delegations with no service layer or transformation:

```go
func (h Handler) CustomerGetOrders(ctx context.Context, request CustomerGetOrdersRequestObject) (CustomerGetOrdersResponseObject, error) {
    orders, err := h.readModel.ListCustomerOrders(ctx, request.Params.CustomerUUID)
    if err != nil {
        return nil, err
    }
    return CustomerGetOrders200JSONResponse{Orders: orders}, nil
}
```

All four handler methods follow this shape. The real work is in the SQL queries and the ReadModel mapping.

We've said it before, but it bears repeating: there's no need to route reads through domain types or service layers. Many developers overcomplicate this by mapping read-only queries through the same layers they use for writes, adding complexity for zero benefit. For reads, a SQL query and a mapping function is all you need.

## Available Orders

That said, there's one query that stands out.

The first three queries are straightforward SELECT with JOIN and WHERE that you've written before. In contrast to the other three, available orders needs to filter by the courier's city. An order is "available" when:

- the restaurant has confirmed it (`restaurant_confirmed_at IS NOT NULL`),
- no courier has been assigned yet (`courier_uuid IS NULL`),
- it hasn't been delivered (`delivered_at IS NULL`),
- and the delivery address city matches the courier's registered city.

The last condition is the non-obvious part. The courier's city is in the `couriers` table, and the delivery city is stored as JSON in the `delivery_address` column of `orders`. You need to extract the city from JSON using the `->>` operator and compare it against a value looked up via subquery.

The available-orders query has more going on, but it's the only one with real complexity.

{{tip}}
These listing endpoints skip pagination for simplicity. In a production system, you'd add LIMIT/OFFSET or cursor-based pagination. It's not very efficient, but for a training exercise it does the job.
{{endtip}}

## Exercise

Exercise path: ./project

Add four order listing endpoints. We prepared the OpenAPI spec and [component tests](https://academy.threedots.tech/knowledge/component-test) for you.

Your `ReadModel` interface in `backend/orders/api/http/handler.go` should gain four new methods:

```go
ListCustomerOrders(ctx context.Context, customerUUID app.CustomerUUID) ([]CustomerOrder, error)
ListRestaurantOrders(ctx context.Context, restaurantUUID app.RestaurantUUID) ([]RestaurantOrder, error)
ListAssignedCourierOrders(ctx context.Context, courierUUID app.CourierUUID) ([]CourierOrder, error)
ListAvailableOrdersForCourier(ctx context.Context, courierUUID app.CourierUUID) ([]CourierOrder, error)
```

1. **`GET /orders/customer/orders`** should return 200 with a list of `CustomerOrder` objects for the given customer. Customer orders should include `restaurant_name` resolved via JOIN with the restaurants table. Orders should be sorted by most recent first.

2. **`GET /orders/restaurant/orders`** should return 200 with a list of `RestaurantOrder` objects for the given restaurant. No JOIN needed. Restaurants know their own name. Should include `customer_uuid` so the restaurant can identify who ordered.

3. **`GET /orders/courier/current-orders`** should return 200 with a list of `CourierOrder` objects assigned to the given courier. Should include `restaurant_name` via JOIN and `delivery_address` so the courier knows where to go.

4. **`GET /orders/courier/available-orders`** should return 200 with a list of `CourierOrder` objects available for pickup. An order is available when it meets the conditions described above. The city match requires comparing a JSON field in orders against the courier's city from the couriers table.

After a courier accepts an order, that order should no longer appear in available orders. Current orders should reflect lifecycle state: `RestaurantPreparedAt` set after the restaurant marks ready, `PickedUpAt` after pickup, `DeliveredAt` after delivery.

{{hints}}

{{hint 1}}
The available-orders query needs to compare the delivery city (stored as JSON) against the courier's registered city (stored in the couriers table). PostgreSQL's `->>` operator extracts a text value from a JSON field. A subquery can look up the courier's city without a JOIN:

```sql
WHERE
    orders.restaurant_confirmed_at IS NOT NULL AND
    orders.courier_uuid IS NULL AND
    orders.delivered_at IS NULL AND
    (orders.delivery_address ->> 'city') = (
        SELECT city
        FROM orders.couriers
        WHERE couriers.courier_uuid = $1
    )
```

The `->>` extracts the `city` field from `delivery_address` as text. The subquery retrieves the courier's registered city. It's not the only way to solve this, but it's straightforward and keeps the filtering in SQL where it belongs.
{{endhint}}

{{endhints}}
