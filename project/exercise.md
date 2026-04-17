# Custom OpenAPI Types

We're writing in a strongly typed language, but we don't take full advantage of it yet.

Right now, `CustomerUUID` is `openapi_types.UUID` under the hood.
If you add more UUID fields later, the compiler can't tell them apart.
And you don't want to pass the customer UUID where an order UUID is expected.

Similarly, the `CountryCode` is a plain `string`, which means it accepts any value.
We can catch it later with some validation, but if we use a custom type, the generated code will do it for us.

[oapi-codegen](https://academy.threedots.tech/knowledge/openapi) lets you map any type to your own Go type.
**The compiler catches type mismatches for you, and you don't need to manually convert between OpenAPI types and your own types in the handler.** Let's set that up.

## Custom Types in OpenAPI

oapi-codegen supports two extension fields that let you override the generated type: **`x-go-type`** names the Go type to use, and **`x-go-type-import`** provides the import path.

For `CustomerUUID` schema in `backend/orders/api/http/openapi.yaml`, it could look like this:

```yaml
    CustomerUUID:
      type: string
      format: uuid
      description: UUID of a customer
      x-go-type: common.UUID
      x-go-type-import:
        path: eats/backend/common
```

The same pattern works for `CountryCode`. After regenerating, the generated type aliases change from this:

```go
type CountryCode = string
type CustomerUUID = openapi_types.UUID
```

To this:

```go
type CountryCode = shared.CountryCode
type CustomerUUID = common.UUID
```

The OpenAPI schema still says `type: string`, but the generated Go code now uses your custom types.
You can use them directly in the request and response structs.

## The UUID Type

Take a look at the new `backend/common/uuid.go` file. `type UUID [16]byte` wraps the same underlying type as `google/uuid`. It implements `MarshalText`/`UnmarshalText` for JSON and `Scan`/`Value` for databases (we'll use those in the next module).

The `NewUUIDv7()` function replaces `uuid.New()`. New UUIDs are generated with `common.NewUUIDv7()` instead of `uuid.New()`.

{{tip}}

Why not stick with v4? `uuid.New()` generates fully random UUIDs. That works fine as an identifier, but it creates a problem you won't notice until the table grows: **inserts get slower over time.**

The database stores rows ordered by primary key in a B-tree index. Random UUIDs land all over the tree, so each insert may touch a different page. As the table grows, more of those pages fall out of memory, and the database has to read from disk to find where to put the new row.

UUID v7 ([RFC 9562](https://www.rfc-editor.org/rfc/rfc9562)) fixes this by putting a timestamp in the first 48 bits. **New IDs are always larger than old ones, so inserts append to the end of the index instead of scattering across it.** You get the same insert performance as auto-increment integers, but without a central counter. Each service instance can generate IDs independently.

How much does this matter in practice? In a [PostgreSQL benchmark](https://www.dujinfang.com/2023/09/02/uuidv4-and-uuidv7.html) with 10 million rows, inserting with a UUID v7 primary key took about 36 seconds. Inserting with UUID v4 took over 4 minutes.

A [MySQL benchmark by Percona](https://www.percona.com/blog/store-uuid-optimized-way/) showed the random UUID table growing almost 50% larger than the ordered one. The gap widens as tables grow, because random UUIDs cause more and more cache misses.

{{endtip}}

## Shared Types

The `backend/common/shared/` package holds types shared between modules. We have only one module right now, so adding shared types feels premature. But setting up the pattern now avoids refactoring every module later when you add a second one.

But it's not that easy. **Shared types are a double-edged sword.** Every type you add creates coupling between every module that uses it. When you change a type in `backend/common/shared`, every team that owns a module using it needs to be involved. Updating `CountryCode` should be fine, but updating a `CustomerProfile` struct with 15 fields can create a cross-team bottleneck.

Keep shared types small:

| Good for shared types                       | Bad for shared types                           |
|---------------------------------------------|------------------------------------------------|
| `UUID` - tiny, stable, universal            | `Customer` struct - [entity](https://academy.threedots.tech/knowledge/entity) owned by one module |
| `CountryCode` - small enum, cross-module    | `OrderStatus` - only one module's concern      |
| `Address` - simple value, no business logic | Database row structs - coupled to schema       |
| `Currency` - fixed set, used in prices      | Request/response structs - API-layer concern   |

For most projects, small data types like these are all you need in shared code. The `SharedTypes` variable in `backend/common/shared/shared.go` registers these types for use in test comparisons across modules.

{{tip}}

Avoid sharing types that serve as inter-module communication contracts (we'll cover inter-module communication later). And avoid database models. Changing one module's schema shouldn't force changes in another.

{{endtip}}

## The Enum Pattern

```go
type CountryCode struct {
    Enum[CountryCodeType]
}

type CountryCodeType string

func (c CountryCodeType) Values() []string {
    return []string{"US", "DE", "GB", "JP", "PL"}
}
```

The pattern uses the **`Enumerable`** interface: any type that declares a `Values() []string` method listing its valid values. `Enum[T Enumerable]` is a generic struct in `backend/common/enum.go` that wraps a string and validates it against `T.Values()` during unmarshaling. If someone sends `"XX"` as a country code, `UnmarshalText` rejects it.

`CountryCode` embeds `Enum[CountryCodeType]` and gets all serialization methods for free: `MarshalText`, `UnmarshalText`, `Scan`, `Value`.

Notice that the `value` field is unexported. You can't create an `Enum` value without going through `UnmarshalText`, which validates against `Values()`. **If you use a `CountryCode` in code, it's guaranteed to be one of the allowed values (or empty).**

To create a new enum:

1. Define a type (e.g., `type OrderStatusType string`)
2. Implement `Values()` returning all valid strings
3. Create a wrapper struct embedding `Enum[OrderStatusType]`

We wrote about this pattern in [Safer Enums in Go](https://threedots.tech/post/safer-enums-in-go/). The code here is the next iteration of that approach, now using generics.

{{tip}}

The `MustEnum` helper function uses an advanced generic constraint to create enum values in one call: `MustEnum[CountryCode]("US")`. You don't need to understand the syntax to use it.

{{endtip}}

## Exercise

Exercise path: ./project

1. Add `x-go-type` and `x-go-type-import` for `CustomerUUID` in `backend/orders/api/http/openapi.yaml`, mapping it to `common.UUID` from `eats/backend/common`.
2. Do the same for `CountryCode`, mapping it to `shared.CountryCode`.
3. Regenerate: run `task gen` (or `go generate ./...`).
4. In `backend/orders/api/http/handler.go`, change `uuid.New()` to `common.NewUUIDv7()` and update the import from `github.com/google/uuid` to `eats/backend/common`.

{{hints}}

{{hint 1}}

The `x-go-type` and `x-go-type-import` fields go directly under the schema definition, at the same indentation level as `type` and `description`. The `x-go-type` value is the qualified Go type name (e.g., `common.UUID`), not the full import path.

{{endhint}}

{{hint 2}}

For `CountryCode` in `backend/orders/api/http/openapi.yaml`:

```yaml
    CountryCode:
      type: string
      description: Country code in ISO 3166-1 alpha-2 format
      x-go-type: shared.CountryCode
      x-go-type-import:
        path: eats/backend/common/shared
```

{{endhint}}

{{endhints}}
