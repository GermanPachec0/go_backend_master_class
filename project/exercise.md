# Dedicated UUIDs

Can you imagine charging the restaurant instead of the customer for an order?
It could be as easy as passing the restaurant ID where a customer ID is expected.
Both are valid UUIDs, so the compiler has nothing to say. Code review may not catch it either.

**Creating a dedicated UUID type turns this class of bug into a compile error.**
(Sure, tests should catch this too. You never skip tests, right?)

We already have a `CustomerUUID` type defined in `backend/orders/app/customer.go`:

```go
type CustomerUUID struct {
	common.UUID
}
```

This looks like a small change from using `common.UUID` directly, but it gives us two separate benefits.

### Distinct Types

The first benefit is **compile-time safety**. Wrapping `common.UUID` in a struct creates a new type that the compiler treats as incompatible with any other UUID.
In the next modules, you'll add `RestaurantUUID`, `RestaurantMenuItemUUID`, and `QuoteUUID`. With distinct types, code like this won't compile:

```go
func GetCustomer(id CustomerUUID) { ... }

// ...
GetCustomer(restaurantUUID)
// compile error: cannot use restaurantUUID (variable of type RestaurantUUID) as CustomerUUID
```

If we used a type alias (`type CustomerUUID = common.UUID`) instead, both `CustomerUUID` and `RestaurantUUID` would be interchangeable with `common.UUID`. The compiler would accept the wrong UUID without complaint. You'd only find out at runtime, when the query returns no rows or, worse, returns the wrong [entity's](https://academy.threedots.tech/knowledge/entity) data.

### UUID Methods

The second benefit comes from *embedding* specifically. We use:

```go
type CustomerUUID struct {
    common.UUID
}
```

Instead of the simpler:

```go
type CustomerUUID common.UUID
```

Both create a distinct type, but the second one doesn't have access to all the UUID's methods.

That means `CustomerUUID` would lose `MarshalText`, `UnmarshalText`, `Value`, `Scan`, `String`, and `IsZero`. Without those methods, JSON serialization and database drivers would break. We'd have to re-implement them separately for each UUID.

With embedding, you can use all methods from `common.UUID` in `CustomerUUID`. **Marshaling for HTTP and database persistence work out of the box.**

{{conversation "From a Past Code Review"}}

{{message "robert"}}

Should `CustomerUUID` validate the value in the constructor? What about empty or malformed UUIDs?

{{endmessage}}

{{message "milosz"}}

For constructors that create *new* UUIDs, `NewUUIDv7()` already guarantees a valid value. When loading from the database, we trust the data. It was valid when it was written.

If we added strict validation on read, we couldn't load historical data when validation rules change. Strict validation belongs at system boundaries: user input and external APIs.

{{endmessage}}

{{message "robert" "milosz:+1" }}

So validate on the way in, trust on the way out of our own storage. Makes sense.

Let's keep an eye on this as we add more types. For some, validating data while loading may be worth the tradeoff.

{{endmessage}}

{{endconversation}}

### Setting It Up

Both [oapi-codegen](https://academy.threedots.tech/knowledge/openapi) and sqlc generate Go code that currently uses `common.UUID` for the customer UUID field. We need to tell them to use `app.CustomerUUID` instead.

It's a few lines of configuration in each tool. The `backend/orders/adapters/db/sqlc.yaml` file already has column overrides (you've seen the one for `address`).
The `backend/orders/api/http/openapi.yaml` file already has `x-go-type` mappings. The pattern is the same.

Once you update both configs and regenerate, the generated code uses `app.CustomerUUID` everywhere, and the compiler will guide you to the remaining spots in the handler and test.

{{tip}}

The `sqlc.yaml` file has both `db_type` overrides (which apply to all columns of that type) and `column` overrides (which apply to a specific column). **A column-level override takes precedence over a `db_type` override.** The existing `uuid` db_type maps all UUID columns to `common.UUID`. Your new column override for `orders.customers.customer_uuid` will override it for that specific column only.

{{endtip}}

## Exercise

Exercise path: ./project

Finish setting up support for the `CustomerUUID`.

1. In `backend/orders/adapters/db/sqlc.yaml`, add a column override that maps `orders.customers.customer_uuid` to `app.CustomerUUID` (with import path `eats/backend/orders/app`).
2. In `backend/orders/api/http/openapi.yaml`, update the `CustomerUUID` schema's `x-go-type` from `common.UUID` to `app.CustomerUUID` and change the import path to `eats/backend/orders/app`.
3. Regenerate code with `task gen`.
4. Update `backend/orders/api/http/handler.go` and `backend/orders/adapters/db/customer_repo_test.go` to use the new type when creating a customer UUID. Simply wrap the common UUID with the new type.

    ```go
    customerUUID := app.CustomerUUID{common.NewUUIDv7()}
    ```

{{hints}}

{{hint 1}}

In `backend/orders/adapters/db/sqlc.yaml`, add under `overrides`:

```yaml
- column: "orders.customers.customer_uuid"
  go_type:
    import: "eats/backend/orders/app"
    type: "CustomerUUID"
```

{{endhint}}

{{hint 2}}

In `backend/orders/api/http/openapi.yaml`, update the `CustomerUUID` schema:

```yaml
x-go-type: app.CustomerUUID
x-go-type-import:
  path: eats/backend/orders/app
```

{{endhint}}

{{endhints}}
