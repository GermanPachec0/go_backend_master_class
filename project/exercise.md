# Generate sqlc

Our database schema is ready. Now you need a way to query it from Go.

Most Go ORMs follow the Rails or Django model: define a struct, and the tool generates SQL for you.

[sqlc](https://docs.sqlc.dev) works in reverse. You write SQL queries, and sqlc generates type-safe Go code. There's no reflection or `any` types, and if a query doesn't match the schema, you get an error at generation time, not at runtime.

It may not sound like the best idea, but it works well. You write the SQL you'd write anyway, annotate it with a comment, and sqlc produces Go functions with exact argument and return types.

Very often, your writes differ from your reads. An INSERT touches five columns in one table, while a dashboard query joins ten tables and returns some aggregated data. sqlc gives you separate types for each operation, which is easy to work with.

The pattern is the same one you've already used with [OpenAPI](https://academy.threedots.tech/knowledge/openapi): write the spec, run `go generate`, and get type-safe code. With OpenAPI, the spec was a YAML file describing HTTP endpoints. With sqlc, the spec is a `.sql` file with annotated queries.

sqlc reads your migration files to infer the schema. It doesn't need a running database.

### The Two Key Files

**`backend/orders/adapters/db/sqlc.yaml`** configures the generator. Take a look at the configuration:

```yaml
version: "2"
sql:
  - engine: "postgresql"           # database engine
    queries: "queries"             # directory with .sql query files
    schema: "migrations"           # migration files (sqlc parses these for schema)
    gen:
      go:
        package: "dbmodels"        # Go package name for generated code
        out: "dbmodels"            # output directory
        sql_package: "pgx/v5"      # use pgx/v5 (not database/sql)
        emit_empty_slices: true    # return []T{} instead of nil for :many queries
        emit_pointers_for_null_types: true  # use *T for nullable columns
        overrides:                 # custom type mappings (see below)
          # ...
```

For most projects, these defaults work fine. See the full [sqlc config reference](https://docs.sqlc.dev/en/latest/reference/config.html) for all available options.

We'll keep the queries in **`backend/orders/adapters/db/queries/*.sql`** files with comment annotations.

Here's the format:

```sql
-- name: InsertAuthor :exec
INSERT INTO authors (id, name, created_at) VALUES ($1, $2, $3);

-- name: GetAuthor :one
SELECT * FROM authors WHERE id = $1;
```

The annotation `-- name: QueryName :command` tells sqlc what to generate. `:exec` produces a function that returns only an `error` (for INSERT, UPDATE, DELETE). `:one` produces a function that returns a single struct and an `error`. There's also `:many` for queries returning multiple rows. See the [query annotations reference](https://docs.sqlc.dev/en/latest/reference/query-annotations.html) for the full list.

sqlc generates separate structs for each query. `InsertCustomer` gets its own `InsertCustomerParams`, while `GetCustomerByUUID` returns an `OrdersCustomer`. In this exercise, they have the same fields.

In practice, you often write different data than you read. This reflects **[CQRS](https://academy.threedots.tech/knowledge/cqrs)** (Command Query Responsibility Segregation): write models and [read models](https://academy.threedots.tech/knowledge/read-model) don't have to match. For more, see [How to use basic CQRS in Go](https://threedots.tech/post/basic-cqrs-in-go/).

{{tip}}

If you use auto-incremented IDs, combine `RETURNING` with `:one` to get the generated ID:

```sql
-- name: CreateAuthor :one
INSERT INTO authors (name) VALUES ($1) RETURNING id;
```

Use `RETURNING *` to get the full row back. You'll see this pattern in a later exercise.

We use UUIDs in this training. Sequential IDs leak how many entities you have (a competitor can estimate your order volume from an order ID), and they require a single sequence that becomes a bottleneck in distributed systems.

{{endtip}}

### Type Mapping

By default, sqlc uses the database driver's types, so we would need to convert the HTTP request types to pgx types.
We want to use our shared types instead: `common.UUID` and `shared.Address`.

The `overrides` section in `sqlc.yaml` handles this with two kinds of overrides:

- **`db_type` overrides** apply globally. Setting `db_type: "uuid"` to `common.UUID` means every `uuid` column in every table uses `common.UUID`. This is the right choice for types that should always map the same way.

- **`column` overrides** target one specific column. The `address` column is stored as `json` in PostgreSQL, but not every `json` column is an address. So we use `column: "orders.customers.address"` to map only that column to `shared.Address`.

You must configure type overrides explicitly for each custom type. sqlc doesn't infer types from your Go code.

Here, `common.UUID` is the same type used in the OpenAPI-generated HTTP code.
**The same UUID flows from the HTTP request through to database insert, so you don't need to map it manually.**

We will need to map the address type, though. It's more complex so it's pragmatic not to couple it to the HTTP code.
We'll look into it in the next exercise.

{{tip}}

Sharing types across layers works well for stable, universal types like `UUID` or `Currency`. For types that evolve differently between layers, keep separate models and write explicit mapping functions. See [When to avoid DRY](https://threedots.tech/post/things-to-know-about-dry/) for more on this trade-off.

{{endtip}}

### Running the Generator

The file `backend/orders/adapters/db/sqlc.go` contains a [Go generate directive](https://pkg.go.dev/cmd/go#hdr-Generate_Go_files_by_processing_source):

```go
//go:generate go tool sqlc generate
```

The `go tool` syntax uses Go 1.24+ [tool dependency management](https://go.dev/doc/modules/managing-dependencies#tools). The exact sqlc version is pinned in `go.mod`, so everyone on the team runs the same version.

You can run this directly with `go generate ./...`, or use `task gen` which runs code generation across all modules. From now on, **every time you add or change a query, run `task gen`.**

**Never edit generated files directly.** They'll be overwritten on the next generation. The `.gitattributes` file already marks `dbmodels/**.go` as generated, so they're collapsed in GitHub PRs. See {{exerciseLink "the .gitattributes exercise" "03-http" "03-gitattributes"}} for more on this pattern.

## Exercise

Exercise path: ./project

Add two queries to `backend/orders/adapters/db/queries/customers.sql`:

1. **`InsertCustomer`** - an `INSERT` with the `:exec` annotation. It should insert a new row into the `customers` table with all five columns: `customer_uuid`, `name`, `email`, `address`, `phone_number`.
2. **`GetCustomerByUUID`** - a `SELECT *` with the `:one` annotation. Select the customer by `customer_uuid`.

Use the PostgreSQL placeholders for query parameters:

```sql
INSERT INTO ... VALUES ($1, $2, $3, $4, $5);

SELECT * FROM ... WHERE ... = $1;
```


Then generate the code by running `task gen` (or `go generate ./...`).

After generation, verify that `backend/orders/adapters/db/dbmodels/customers.sql.go` exists with `GetCustomerByUUID` and `InsertCustomer` methods. The type mapping is already configured in `sqlc.yaml`. We'll use the generated code in the next exercise.

{{hints}}

{{hint 1}}

Use the schema-qualified table name `orders.customers` (matching your migration).

{{endhint}}

{{hint 2}}

```sql
-- name: InsertCustomer :exec
INSERT INTO
    orders.customers (
    customer_uuid,
    name,
    email,
    address,
    phone_number)
VALUES
    ($1, $2, $3, $4, $5)
;
```

{{endhint}}

{{hint 3}}

```sql
-- name: GetCustomerByUUID :one
SELECT
    *
FROM
    orders.customers
WHERE
    customer_uuid = $1
;
```

{{endhint}}

{{endhints}}
