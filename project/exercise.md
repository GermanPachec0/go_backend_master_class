# Implement Repository

In the previous exercise, we put SQL calls directly in the handler. For a single endpoint, that's good enough.

But as the project grows, handlers start mixing too many concerns.
It's difficult to understand the logic next to the database calls.
If two endpoints use the same database query, you have to duplicate it.
When writing tests, you have to run the server and call the HTTP endpoints.

That's where the **[Repository Pattern](https://academy.threedots.tech/knowledge/repository-pattern)** comes in.
The idea is to separate database logic from the rest of the application.
Your HTTP handler doesn't need to know about sqlc queries or `InsertCustomerParams`.
It calls a method on the repository, and the repository handles the database details.

**Keeping the logic of your application together with your database logic makes your application much more complex, harder to test, and harder to maintain.**
With a repository, you can test business logic by mocking the repository interface.
You can change the database implementation without touching handlers.
And when two handlers need to insert a customer, the logic lives in one place.

### Integration Tests

We prepared the boilerplate for you.
Take a look at `CustomerRepository` in `backend/orders/adapters/db/customer_repo.go`.

There are also integration tests in `backend/orders/adapters/db/customer_repo_test.go`.
They verify that `RegisterCustomer` correctly persists customer data to a real PostgreSQL database.

{{tip}}

**[Integration tests](https://academy.threedots.tech/knowledge/integration-testing)** test your adapter in isolation from the rest of the application, but with real infrastructure (like a real database).

|                      | Unit Tests         | Integration Tests             |
|----------------------|--------------------|-------------------------------|
| Needs infrastructure | No                 | Yes                           |
| Execution speed      | Fast               | Fast (seconds)                |
| What they test       | Logic in isolation | Adapter + real infrastructure |
| Mocks                | Most dependencies  | Usually none                  |

For more on where integration tests fit in the bigger picture, see [Microservices test architecture](https://threedots.tech/post/microservices-test-architecture/).

{{endtip}}

You can run the tests locally with `task test-integration` (or `go test -tags integration ./...`).
The build tag is there to prevent these tests from running in regular `task test` (or `go test ./...`) since they require a real database and are a bit slower than unit tests.

The tests spin up a real PostgreSQL database, run migrations, and test the repository method directly.
This catches real infrastructure issues that unit tests miss (like SQL syntax errors or mismatched field types).
They're still fast enough to run frequently during development.

{{tip}}

Notice how the test uses [`cmp.Diff`](https://pkg.go.dev/github.com/google/go-cmp/cmp) to compare the expected and actual customer. **`cmp.Diff` is better than comparing field by field manually.** If you add a new field to the struct, the test will fail automatically because you didn't set the new field in the expected value. With field-by-field comparison, you'd likely forget to update the test.

There's one gotcha, though: the test passes `cmpopts.EquateComparable(shared.SharedTypes...)` as an option. Types like `CountryCode` contain unexported fields (from the `Enum[T]` embedding), and `cmp.Diff` panics when comparing structs with unexported fields by default. [`EquateComparable`](https://pkg.go.dev/github.com/google/go-cmp/cmp/cmpopts) tells go-cmp to use Go's `==` operator for those types instead of inspecting their internals. The `shared.SharedTypes` slice lists all types that need this.

We usually don't recommend `cmp.Diff` for production code since it relies on reflection. For tests, it's a great choice. You'll also see `cmpopts.SortSlices` in later exercises for comparing lists where order doesn't matter.

{{endtip}}

## Exercise

Exercise path: ./project

Implement the `RegisterCustomer` method in `backend/orders/adapters/db/customer_repo.go`.

Use the sqlc-generated `InsertCustomer` method exactly like in the HTTP handler before.

You don't need to integrate the repository into the handler yet. We'll do that in the next exercise. For now, make the integration tests pass.

You can run them locally like this (this is optional):

- Run docker compose with `task up` or `docker compose up`
- In another terminal, run `task test-integration` or `go test -tags integration ./...`

{{tip}}

We execute a single query here, so no explicit transaction is needed. A single `INSERT` is atomic on its own. We'll introduce transactions in the Advanced Repositories module when operations span multiple queries.

You might notice that `dbmodels.New()` accepts a `DBTX` interface, not a concrete type. This will let us use it with transactions later.

{{endtip}}

{{hints}}

{{hint 1}}

A good starting point is simply copying what your HTTP handler currently does, then adjusting the code to fit the repository method signature.

{{endhint}}

{{endhints}}
