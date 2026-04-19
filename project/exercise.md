# SQL Migrations

The register customer endpoint is ready to use by the frontend app, but it doesn't do anything yet.
Let's prepare a way to store the customer's data. We're using PostgreSQL as our database.

We'll start with SQL migrations.

## How Migrations Work

SQL migrations are **versioned, incremental changes** to your database schema. Each migration is a numbered SQL file that gets applied in order.
When you start the application locally, all migrations run from the first one.

**In production, only new migrations that haven't been applied yet are executed.** This way, each environment stays in sync, and you don't run any manual SQL scripts.

We use the [migrate](https://github.com/golang-migrate/migrate) library in our project. It's widely adopted, has a minimal API, and works well with Go's [`embed`](https://pkg.go.dev/embed) package.

## Where Migrations Live

Migration files go in `backend/orders/adapters/db/migrations/`.

Adapters are the second *layer* we introduce. They contain all the code that interacts with external systems like databases, HTTP APIs, the filesystem, etc. The `db` adapter contains all database-related code for the orders module.

The path may look nested, but it's on purpose.
As the project grows with more modules, each module's migrations stay isolated. Remember, we're building a modular monolith where each module could be owned by a different team.

In smaller projects with one schema, keeping all migrations in a single top-level directory is fine. Use what works for your project.

## How Migrations Run in This Project

In our project, **migrations are embedded directly in the binary** and run on application startup. You don't need migration scripts, sidecar containers, or extra CI steps.

Take a look at `backend/orders/module.go`:

```go
//go:embed adapters/db/migrations/*.sql
var embedMigrations embed.FS
```

The `//go:embed` directive tells the Go compiler to bundle all `.sql` files from `adapters/db/migrations/` into the binary at build time. At runtime, `embedMigrations` acts as a read-only filesystem.
In practice, this means **you can distribute the application as a single binary file**, and don't need to worry about copying the SQL files separately.

The `Init` function calls `MigrateDatabaseUp` (defined in `backend/common/migrations.go`) with the embedded files:

```go
func (m *Module) Init(ctx context.Context) error {
	// ...
	if err := common.MigrateDatabaseUp(
		ctx,
		string(m.Name()),
		m.stdDb,
		embedMigrations,
		"adapters/db/migrations",
	); err != nil {
		return err
	}
	// ...
}
```

**To add a new migration**, create a new `.up.sql` file with the next sequential number (e.g., `0002_add_orders_table.up.sql`). The embed's glob pattern (`*.sql`) picks it up automatically.

{{tip}}

There are two common naming strategies for migration files: sequential numbers (`0001_`, `0002_`) and timestamps (`20260218120000_`).

In this training, we'll use sequential numbers.

With timestamps, when two PRs add migrations in parallel, they can be merged in any order, and you don't control which one runs first in production.
This can cause schema drift between the production and your local or CI environment, leading to hard-to-debug issues.

Sequential numbers have their own downside: if two people add a migration at the same time, they'll pick the same number and one PR will need a rename after the other merges.
That's a minor inconvenience compared to unpredictable ordering in production, though.

{{endtip}}

## PostgreSQL Schemas

**Each module in our project gets its own PostgreSQL schema.** A schema is a namespace within a database. `orders.customers` means the `customers` table in the `orders` schema.

As the project grows with more modules, this gives clear ownership boundaries and prevents table name collisions. You can still join across schemas when needed.

You might wonder why `CREATE SCHEMA IF NOT EXISTS orders` appears in two places: in the migration file and in `MigrateDatabaseUp()` in Go code. Both are necessary.

The Go function creates the schema at runtime before migrations execute, because `migrate` needs the schema to exist so it can create its `schema_migrations` tracking table.

The migration file needs it because of how the sqlc library works (you'll learn about it in the next module).

Both use `IF NOT EXISTS`, so there's no conflict at runtime.

## Migration Rules

**Use sequential numbers for migrations.** Once a migration is merged, don't change it. Create a new one instead.

Write migrations that work with both old and new code versions. Your CI should catch failing migrations before they reach production.

There two kinds of migrations: up (applying changes) and down (reverting changes).

We write **up migrations only**.

Using down migrations in production is tricky and risky. It's better to fix forward with a new migration.

Locally, it's easier to clean the database and start from scratch than to think about which version to roll back to. No need to write something you'll never use. Down migrations are possible if you want them. See the [migrate docs](https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md) for the file naming convention.

**Wrap your migrations in an explicit transaction** (`BEGIN` / `COMMIT`). PostgreSQL supports transactional schema changes, but `migrate` doesn't auto-wrap your file in a transaction. Without explicit wrapping, a failing migration with multiple statements could leave the schema in a partially applied state.

Even with a single `CREATE TABLE`, wrapping in a transaction is a good default. It costs nothing and prevents surprises if you decide to extend the migration later.

## Exercise

Exercise path: ./project

{{tip}}

If you mess something up with migrations locally, run `task up-clean` to start from a clean state.
When running `tdl tr run`, you always start with a clean database.

{{endtip}}

Implement the migration in `backend/orders/adapters/db/migrations/0001_init_orders.up.sql`.

This migration should create the `orders.customers` table with these columns:

| Column          | Type           | Constraints               |
|-----------------|----------------|---------------------------|
| `customer_uuid` | `uuid`         | `NOT NULL`, `PRIMARY KEY` |
| `name`          | `varchar(255)` | `NOT NULL`                |
| `email`         | `varchar(255)` | `NOT NULL`                |
| `address`       | `json`         | `NOT NULL`                |
| `phone_number`  | `varchar(50)`  | `NOT NULL`                |

Remember to wrap the migration in a transaction (`BEGIN` / `COMMIT`) and create the schema before the table (`CREATE SCHEMA IF NOT EXISTS orders`).

{{conversation "From a Past Code Review"}}

{{message "milosz"}}

I noticed you used `customer_uuid` instead of just `id` for the primary key. Any particular reason?

{{endmessage}}

{{message "robert" "milosz:+1"}}

When you have bare `id` columns, every JOIN needs aliasing, like: `o.id`, `c.id`, `oi.id`, etc.
With named keys like `customer_uuid`, the column is self-documenting. You can grep for it and immediately know what it refers to.

Plus, PostgreSQL's `USING` clause works cleanly: `JOIN order_items USING (order_uuid)`.

{{endmessage}}

{{endconversation}}

{{tip}}

You can verify your migration by connecting to the database with `task pgcli` (install from [pgcli.com](https://www.pgcli.com/install)) after running `task up`. You can also use any database client of your choice with connection string `postgres://user:password@localhost:5432/eats`.

Remember to call `SET search_path TO 'orders';` to work with the orders schema.

{{endtip}}

{{hints}}

{{hint 1}}

Your migration file needs three structural elements:

1. Wrap everything in a transaction (`BEGIN` at the start, `COMMIT` at the end)
2. Create the schema: `CREATE SCHEMA IF NOT EXISTS orders;`
3. Create the table with the `orders.` prefix: `CREATE TABLE orders.customers (...)`

Use the column spec table above for the column names, types, and constraints.

{{endhint}}

{{hint 2}}

Here's one way to implement this:

```sql
BEGIN;

CREATE SCHEMA IF NOT EXISTS orders;

CREATE TABLE orders.customers
(
    customer_uuid uuid         NOT NULL,
    name          varchar(255) NOT NULL,
    email         varchar(255) NOT NULL,
    address       json         NOT NULL,
    phone_number  varchar(50)  NOT NULL,
    PRIMARY KEY (customer_uuid)
);

COMMIT;
```

{{endhint}}

{{endhints}}
