# Component Tests

We already have [integration tests](https://academy.threedots.tech/knowledge/integration-testing) for our repositories, covering how adapters talk to the database.

**The next step is [component tests](https://academy.threedots.tech/knowledge/component-test): verifying the full request lifecycle with the HTTP API.**
You start the real service, connect to a real database, send an HTTP request, and check what comes back.

## Component Tests vs Other Tests

Component tests sit in a sweet spot between unit and end-to-end tests.
They verify complete business cases through your public API (HTTP), using real infrastructure, while mocking only external services:

| Feature / Test Type           | Unit                       | Integration  | Component        | End-to-End |
|-------------------------------|----------------------------|--------------|------------------|------------|
| **Needs infrastructure**      | No                         | Yes          | Yes              | Yes        |
| **Use of external systems**   | No                         | No           | No               | Yes        |
| **Focused on business cases** | Depends on the tested code | No           | Yes              | Yes        |
| **Uses mocks or stubs**       | Most dependencies          | Usually none | External systems | None       |
| **Tested API**                | Go package                 | Go package   | HTTP             | HTTP       |
| **Execution speed**           | Fast                       | Fast         | Medium           | Slow       |

**Component tests cover the most ground per test.** If a component test passes, you know the handler, the [application layer](https://academy.threedots.tech/knowledge/application-service), and the [repository](https://academy.threedots.tech/knowledge/repository-pattern) all work together.
Unit tests can be added later for edge cases and complex logic that component tests can't easily reach.

Because you mock external systems, running component tests is much easier than end-to-end tests.
You can test your service in isolation even if another service is down or has a broken contract.

Your aim is to have a component test for each critical path in your application.

## Independent Tests

{{conversation "From a Past Code Review"}}

{{message "milosz"}}

I see you added `t.Parallel()` to the test. Is it worth the effort for just one test?

{{endmessage}}

{{message "robert" "milosz:+1"}}

Adding it now costs nothing and sets the right pattern. Without it, every new test runs sequentially, and the suite gets slow fast. More importantly, parallel tests force you to keep data isolated. If a test only works sequentially, it usually means it depends on shared state, and that's a bug waiting to happen.

{{endmessage}}

{{endconversation}}

Each test should be independent and use `t.Parallel()`.

It's possible by using random data (like `testutils.GenerateRandomCountry()`).

Every test creates its own data, so tests don't conflict even when running against the same database.

The naive approach to independent tests is adding `t.Cleanup()` to delete modified rows or running `TRUNCATE TABLE` to reset the database.
Don't do that. Tests like that are fragile and hard to maintain.
As the suite grows, it'll be more painful to understand how the tests interact with each other and why they fail.

**Each test should insert unique data and check only that data.**
This way, you avoid shared fixtures, ordering dependencies, and flaky failures from leftover state.

Because tests don't share data, they can run in parallel without interfering with each other.
You get a fast feedback loop that stays reliable as the test suite grows.

## `assert` vs `require`

We use [`testify`](https://github.com/stretchr/testify) for assertions. It cuts a lot of boilerplate and offers two packages: `assert` and `require`.

They work in a similar way, but **`require` stops the test immediately on failure. `assert` records the failure but lets the test continue.**

Take a look at how `registerCustomerInCity` uses `require`.
If the HTTP call fails, `require` stops the test with a clear message like "expected 201, got 500."

{{codeFile "backend/tests/helpers_test.go"}}

```go
resp, err := deps.OrdersApiClients.RegisterCustomerWithResponse(ctx, customerToCreate)
require.NoError(t, err)
require.Equal(t, http.StatusCreated, resp.StatusCode())
require.NotNil(t, resp.JSON201)
```

If you used `assert` instead, the test would keep running after a failed HTTP call, try to read `resp.JSON201` (which is `nil`), and panic with a confusing nil pointer error instead of telling you the real problem.

Use `require` for setup steps where failure means everything downstream is meaningless.
Use `assert` for final assertions where the test is about to end anyway.

All helpers also call `t.Helper()`, so failure messages show the calling test function, not the line inside the helper.

{{tip}}

The [Google Go Style Guide](https://google.github.io/styleguide/go/decisions.html) recommends against assertion libraries.
Testify saves enough boilerplate that it's worth it for us.
We didn't experience any downsides after using it for many years.

{{endtip}}

## How the Test Setup Works

Let's trace through what happens when you run `go test`.

`TestMain` is a special function recognized by `go test`.
It works sort of like the `main` function but for tests.
If present in a test package, Go calls it instead of running tests directly.
You control what happens before and after all tests.

In `backend/tests/setup_test.go`, `TestMain` does the following:

1. Creates the generated [OpenAPI](https://academy.threedots.tech/knowledge/openapi) client pointing to `http://localhost:9090/`.
2. Connects to PostgreSQL and starts the service with `backend.New(ctx, dbPgx, dbStd)`. It's the real service, not a mock.
3. Starts the HTTP server in a goroutine and polls `/health` until ready.
4. Stores the API client in a package-level `deps` variable that all tests access.
5. Calls `m.Run()` to run all tests, then shuts down gracefully.

Once this setup is ready, every component test in the package reuses it.

We also have a bunch of helpers in `backend/tests/helpers_test.go`. They use `gofakeit` to generate random names, emails, and addresses.

{{tip}}

You'll also see gateway-related code in the test dependencies. We'll cover the gateway in a later module. For now, it's there as a placeholder.

{{endtip}}

## Further Reading

- [Database Integration Testing in Go](https://threedots.tech/post/database-integration-testing/) - Fast, parallel database tests with real PostgreSQL in Docker.
- [Go Test Parallelism](https://threedots.tech/post/go-test-parallelism/) - Why `t.Parallel()` matters and common pitfalls.
- [Microservices Test Architecture](https://threedots.tech/post/microservices-test-architecture/) - How component tests fit alongside unit, integration, and e2e tests.

## Exercise

Exercise path: ./project

Write your first component test to verify that registering a customer works end-to-end.

In `backend/tests/component_test.go`, replace `t.Error("TODO")` with a real test:

1. Generate a random country with `testutils.GenerateRandomCountry()`.
2. Call `registerCustomerInCity(ctx, t, country, "Some city")` to register a customer and capture the returned UUID.
3. Assert the customer UUID is not empty using `assert.NotEmpty()`.

You can run tests locally with `task test-component` (make sure Docker is running with `task up` first). This is optional: the `tdl` CLI handles everything for you.

(If you don't use Task, run `go test ./backend/tests/...` and `docker compose up` instead.)

{{hints}}

{{hint 1}}

**Use `t.Context()` for the context argument.**

It's a context that gets canceled when the test times out or is stopped.

{{endhint}}

{{hint 2}}

Here's the full implementation:

```go
ctx := t.Context()
country := testutils.GenerateRandomCountry()
customerUUID := registerCustomerInCity(ctx, t, country, "Some city")
assert.NotEmpty(t, customerUUID)
```

{{endhint}}

{{endhints}}
