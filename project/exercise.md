# Integrate Repository

We have a working `CustomerRepository` covered by [integration tests](https://academy.threedots.tech/knowledge/integration-testing).
But the HTTP handler still uses `*pgxpool.Pool` and talks to the database directly.
Let's connect the [repository](https://academy.threedots.tech/knowledge/repository-pattern) to hide the database behind an interface.

### Injecting the Repository

We need to call the repository from the handler.
First, we need to inject it in the same way we injected the database pool before.
But this time, we'll use an interface instead of a concrete type.

The handler declares what it needs as an interface with a single method: `RegisterCustomer`.

```go
type CustomerRepository interface {
    RegisterCustomer(ctx context.Context, customerUUID common.UUID, customer RegisterCustomer) error
}
```

In some languages, it's common to define interfaces close to the implementation.
**In Go, we define the interface close to where it's used, in the handler file.**

This is because Go interfaces are implemented implicitly, so the repository implementation doesn't need to know about the interface at all.
It also helps us avoid import cycles between packages.

### Why the Interface Lives Here

There's a very practical reason for keeping the interface where it's used: **it avoids import cycles**.

The `db` package imports `http` types (it uses `http.RegisterCustomer` as an argument).
If the interface lived in the `db` package, the `http` package would need to import `db` for the interface, and `db` already imports `http`.
This creates an import cycle, and it won't compile.

{{tip}}

This is a lightweight form of [Clean Architecture](https://threedots.tech/post/introducing-clean-architecture/): the handler defines its dependencies as interfaces, the database adapter implements them, and `backend/orders/module.go` connects the two.
We don't need the full [Clean Architecture](https://academy.threedots.tech/knowledge/clean-architecture) setup here.
It's good enough to keep the interface close to the handler to have a clear dependency flow.

{{endtip}}

### Decoupling

After we inject the repository to the handler, the HTTP handler is shorter and simply calls the repository method.
It still generates the UUIDv7 (that's not a database concern).

```go
func (h Handler) RegisterCustomer(ctx context.Context, request RegisterCustomerRequestObject) (RegisterCustomerResponseObject, error) {
	customerUUID := common.NewUUIDv7()

	err := h.customerRepository.RegisterCustomer(ctx, customerUUID, *request.Body)
 	if err != nil {
		return nil, err
	}

	// ...
```

Now, the handler no longer imports the `db` package.
If you read the handler code and can't tell what database it uses, that's a good sign.
With a simple interface, we decoupled the two layers.

{{tip}}

**What about testing handlers?**
For thin handlers like this one, we don't recommend writing unit tests with mock repositories.
The maintenance cost rarely pays off.
If a handler has complex logic, extract it into helper functions and test those directly.
We'll cover [component tests](https://threedots.tech/post/microservices-test-architecture/) in a later module.
They check the integration of the entire service and catch more real issues.

{{endtip}}

## Exercise

Exercise path: ./project

1. **Define a `CustomerRepository` interface** in `backend/orders/api/http/handler.go` with a `RegisterCustomer` method matching the concrete repository from the previous exercise.
2. Update the `Handler` struct and `NewHandler` to accept this interface instead of `*pgxpool.Pool`.
3. **Update the `RegisterCustomer` HTTP handler** to call the repository's `RegisterCustomer()`.
4. Inside the `Init` method in `backend/orders/module.go`, create the concrete repository with `db.NewCustomerRepository(m.pgxDb)` and pass it to `NewHandler`.

{{hints}}

{{hint 1}}

Your new handler should look like this:

```go
type Handler struct {
	customerRepository CustomerRepository
}

func NewHandler(
	customerRepository CustomerRepository,
) Handler {
	if customerRepository == nil {
		panic("customerRepository cannot be nil")
 	}

	return Handler{
		customerRepository: customerRepository,
	}
}
```

{{endhint}}

{{hint 2}}

Inject the dependency like this:

```go
func (m *Module) Init(ctx context.Context) error {
	customerRepo := db.NewCustomerRepository(m.pgxDb)

	httpHandler := http2.NewHandler(customerRepo)
	// ...
```

{{endhint}}

{{endhints}}
