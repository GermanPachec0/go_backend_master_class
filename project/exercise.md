# Courier Registration

{{message "milosz"}}
Before customers can place orders, someone needs to deliver them. You're adding a `POST /orders/register-courier` endpoint, following the same architectural pattern as customer registration.

A courier has a name, phone number, and city. The city determines which restaurants they can deliver for.

The highlighted elements on the board below are what you'll add in this exercise.
{{endmessage}}

{{miroBoard}}

## What's Already Provided

We've already updated the [OpenAPI](https://academy.threedots.tech/knowledge/openapi) spec to include the `/orders/register-courier` endpoint with `RegisterCourier` and `RegisterCourierResponse` schemas.
There are also new [component tests](https://academy.threedots.tech/knowledge/component-test) for the endpoint.

After you regenerate the server code (with `task gen` or `go generate ./...`), the `RegisterCourier` method appears in `StrictServerInterface` and the new request and response types become available. Your handler needs to implement it.

You'll know when everything works because the tests cover both the happy path and validation errors.

That's your starting point.

## Layers and References

You'll touch every layer: database migration, sqlc query, repository, [application service](https://academy.threedots.tech/knowledge/application-service), HTTP handler, and module wiring. That said, if any of these layers feel rusty:

- {{exerciseLink "Insert Customer" "04-database" "03-insert-customer"}} -- wiring a repository into the handler, [dependency injection](https://academy.threedots.tech/knowledge/dependency-injection)
- {{exerciseLink "Add Migrations" "04-database" "01-add-migrations"}} -- creating SQL migrations
- {{exerciseLink "Generate sqlc" "04-database" "02-generate-sqlc"}} -- sqlc queries, config, and column overrides
- {{exerciseLink "Implement Repository" "05-repository" "01-implement-repository"}} -- repository pattern implementation
- {{exerciseLink "Dedicated UUIDs" "06-application-layer" "02-dedicated-uuids"}} -- dedicated UUID types with struct embedding
- {{exerciseLink "Error Handling" "07-errors-and-testing" "01-error-handling"}} -- validation with error details

## Expected Schema

The schema mirrors the customers table. Your `orders.couriers` table should look like this:

```sql
CREATE TABLE orders.couriers
(
    courier_uuid uuid         NOT NULL,
    name         varchar(255) NOT NULL,
    phone_number varchar(50)  NOT NULL,
    city         varchar(100) NOT NULL,
    PRIMARY KEY (courier_uuid)
);
```

Your migration number is `0005` (it follows `0004_fts_index.up.sql`). All columns are `NOT NULL` because every field is required for a valid courier registration. The `courier_uuid` column is the primary key, same as `customer_uuid` in the customers table.

## Exercise

Exercise path: ./project

**Implement courier registration end-to-end.** First, regenerate the server code so the new endpoint types are available: run `task gen` (or `go generate ./...` if you don't use Task).

Your feature should satisfy these requirements:

- `POST /orders/register-courier` returns HTTP 201 with `courier_uuid` in the JSON response.
- The courier is persisted in the `orders.couriers` table with the correct name, phone number, and city.
- Empty name, phone number, or city returns HTTP 400 with error details.
- A `CourierUUID` dedicated type with a sqlc column override (same pattern as `CustomerUUID`).
- The `RegisterCourier` handler method implements `StrictServerInterface` (generated from the OpenAPI spec).
- A new `CourierRepository` available to the application service.

**The steps should feel familiar.** You're building the same layered structure as customer registration: migration, SQL query, sqlc config update, repository, application service method with validation, HTTP handler, and module wiring. The order of implementation is up to you.

Return error details the same way customer registration does. If a required field is empty, return a `400` with the specific field name in the error details (for example, `"name"` or `"phone_number"`). Both the happy path and validation errors are covered by the provided component tests.
