# Restaurant Browsing

Customers need to browse restaurants and read menus before placing an order. Both endpoints are read-only queries with no domain logic, state transitions, or authorization. It's the same full vertical slice: {{exerciseLink "HTTP handler" "03-http" "01-http-handler"}}, {{exerciseLink "sqlc" "04-database" "02-generate-sqlc"}} query, {{exerciseLink "repository" "05-repository" "01-implement-repository"}}.

In {{exerciseLink "module 06" "06-application-layer" "01-app-types"}}, we moved the repository out of the handler and behind the [service layer](https://academy.threedots.tech/knowledge/application-service). Now we're going the other direction: the handler calls the repository directly for these reads.

That's not a contradiction. **Writes go through the service layer because that's where validation, business rules, and [orchestration](https://academy.threedots.tech/knowledge/orchestration) live. Reads without business logic don't need that indirection.**

A big part of what goes wrong in codebases comes from following rules dogmatically without understanding why they exist. Routing every read through the service layer "for consistency" is exactly that.

The service layer is not only for writes, though. If a read needs authorization, aggregation from multiple sources, or business rules on visibility, the service layer is the right place for it. Saying "never use the service for reads" would be its own kind of dogma. **The point is to understand what each layer is for and be pragmatic about it.**

No new commands or events on the board this time. Instead, the green card appears: a **view** (or [read model](https://academy.threedots.tech/knowledge/read-model)).

On an [Event Storming](https://academy.threedots.tech/knowledge/event-storming) board, green cards represent data that's shown to users or used by commands to make decisions.
They're usually placed next to the event that produces the data they display.
In this case, the Quote view sits next to "Quote Created" because it represents the data saved after a quote is created.

Views don't change state. They represent what's already in the system.

{{miroBoard}}

The handler already has a `RestaurantReader` interface (with `RestaurantName`, used by `CustomerPlaceOrder`). You'll add `ListRestaurants` and `GetRestaurantMenu` to it. The same `RestaurantRepository` struct satisfies both this interface and the write-side `RestaurantRepository` interface that the service uses.

This is the same idea as the {{exerciseLink "read models" "09-read-models" "01-simple-read-model"}} from the previous module. There, the handler owned a `ReadModel` interface and called the database directly, returning HTTP types. Here, the data maps to domain types (`app.Restaurant`, `app.RestaurantMenu`), but the handler still owns the dependency.

**The handler never does writes directly. That rule hasn't changed.**

## Exercise

Exercise path: ./project

Your endpoints should match the [OpenAPI](https://academy.threedots.tech/knowledge/openapi) spec already defined in `backend/orders/api/http/openapi.yaml`. Implement the full vertical slice: SQL query, repository method, handler interface, handler method.

**Add two methods to the `RestaurantReader` interface in `backend/orders/api/http/handler.go`**: `ListRestaurants` and `GetRestaurantMenu`. The interface already has `RestaurantName`. No changes needed in `NewHandler` or `backend/orders/module.go`.

**`CustomerListRestaurants`** (`GET /orders/customer/restaurants`):
- Should return HTTP 200 with a `restaurants` array
- Each restaurant includes `uuid`, `name`, `description`, `address`
- Restaurants sorted by name ascending

**`CustomerGetRestaurantMenu`** (`GET /orders/customer/{restaurant_uuid}/menu`):
- Should return HTTP 200 with `restaurant_uuid`, `restaurant_name`, `description`, `currency`, `address`, and an `items` array
- Each item includes `uuid`, `name`, `gross_price`, `ordering`
- Only active (non-archived) menu items should appear
- The existing `GetRestaurantMenu` and `GetRestaurant` sqlc queries cover this endpoint. No new SQL needed
- Should return HTTP 404 with `NotFoundError` if the restaurant UUID doesn't exist

You need one new SQL query: `ListRestaurants` in `backend/orders/adapters/db/queries/restaurants.sql`. The existing `GetRestaurantMenu` and `GetRestaurant` queries already handle the menu endpoint.
