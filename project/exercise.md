# HTTP Handler

With the project setup in place, we can now add a public API to our service.

We're going to run an HTTP server with endpoints for all the operations.
These endpoints will be called by the frontend web app, mobile apps, and possibly some internal tools.

We'll define the contract as [OpenAPI](https://academy.threedots.tech/knowledge/openapi) schema, generate Go code from it, and expose the first endpoint.

## Project Layout

Take a look at the `backend/orders/api/http` package.
This is where we'll keep OpenAPI definitions and HTTP handlers.

The `api` package represents the entry points to the module. It's how other modules or external clients interact with it.
For now, there's just one entry point: the HTTP API.

The `api` package is the first *layer* of the module. We'll introduce more layers later.

## OpenAPI-First Design

Instead of defining the HTTP endpoints in the code, **we're going to define the OpenAPI spec first, then generate Go code from it.**

If you've used [gRPC](https://academy.threedots.tech/knowledge/grpc) with [Protocol Buffers](https://academy.threedots.tech/knowledge/protocol-buffers), this should feel familiar. Instead of a `.proto` file, we have `backend/orders/api/http/openapi.yaml` that describes our endpoints and data models.

We'll use [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to generate types, interfaces, and routing setup. It handles the marshaling and routing boilerplate, so you can focus on business logic.

**Why OpenAPI-first?**

* The spec becomes your contract. Backend and frontend teams see the same schema.
* API documentation comes for free (tools like Swagger UI read OpenAPI directly).
* You regenerate Go code after each change to the API. Breaking changes show up as compile errors, not at runtime. It helps you catch issues before they reach production.

## The oapi-codegen Config

Take a look at the configuration:

{{codeFile "backend/orders/api/http/oapi-codegen.yaml"}}

```yaml
package: http
generate:
  echo-server: true
  strict-server: true
  models: true
output: openapi.gen.go
```

- `package: http` sets the Go package name for the generated file.
- `echo-server: true` generates code compatible with the Echo library we use.
- `strict-server: true` generates **`StrictServerInterface`**, which enforces a strict contract: each method receives a typed request struct and returns a typed response struct (similar to gRPC handlers generated from protobuf). This is the interface you'll implement.
- `models: true` generates Go structs from the OpenAPI schemas (`Address`, `RegisterCustomer`, `ErrorResponse`, etc.).
- `output: openapi.gen.go` is the output file name.

The `go:generate` command lives in `backend/orders/api/http/openapi.go`:

```go
//go:generate go tool oapi-codegen --config=oapi-codegen.yaml openapi.yaml
```

The `go tool` syntax (Go 1.24+) runs tools declared as dependencies in `go.mod`, so you don't need to install `oapi-codegen` separately.

Run `task gen` from the project root (or `go generate ./...`) to execute this command. It reads `backend/orders/api/http/openapi.yaml` and `oapi-codegen.yaml`, then writes `openapi.gen.go` in the same directory.

## What Gets Generated

After running generation, you'll see a new file: `openapi.gen.go`. It contains hundreds of lines of generated code: request and response types, interfaces, and routing. **Never edit this file.** When the spec changes, just regenerate it.

The key piece is the **`StrictServerInterface`**:

```go
type StrictServerInterface interface {
    RegisterCustomer(ctx context.Context, request RegisterCustomerRequestObject) (RegisterCustomerResponseObject, error)
}
```

That's the interface you need to implement.

As you can see, the exact endpoint signature (`POST /orders/register-customer`) isn't mentioned anywhere.
We simply trust the generated code to route requests to the right method based on the OpenAPI spec.

oapi-codegen also generates response types for each status code defined in the spec. The naming convention is `{OperationID}{StatusCode}JSONResponse`:

- `RegisterCustomer201JSONResponse` for a successful 201 response
- `RegisterCustomer400JSONResponse` for a 400 bad request
- `RegisterCustomer409JSONResponse` for a 409 conflict

Each response type is a Go struct matching the schema defined in the OpenAPI spec.
For example, `RegisterCustomer201JSONResponse` has a `CustomerUuid` field of type `openapi_types.UUID`.
Simply return the proper struct, and the generated code handles JSON serialization and HTTP headers.

The return type `RegisterCustomerResponseObject` is an interface, so the compiler won't let you return a type that doesn't match the spec. **This gives you compile-time safety for your HTTP responses.** If you try to return a 204 No Content response but the spec only defines 201, 400, and 409, the code won't compile.

{{tip}}

You may have noticed the logging exercise uses `shortuuid` for correlation IDs, while here we use `uuid.New()` from `google/uuid`. These serve different purposes: `shortuuid` generates compact, URL-safe strings good for log correlation. For [entity](https://academy.threedots.tech/knowledge/entity) IDs like customer UUIDs, we use standard UUIDs because they're the universal format for database primary keys and APIs.

{{endtip}}

{{tip}}

There are two generated interfaces: `ServerInterface` and `StrictServerInterface`. You want to implement `StrictServerInterface`.

The "strict" version gives you typed request/response objects instead of raw `echo.Context`. It does JSON parsing, validation, and response encoding for you. You get type safety and less boilerplate.

{{endtip}}

## The Starting Point

Right now, `backend/orders/api/http/handler.go` contains an empty `Handler` struct and a `Register` function that does nothing:

```go
type Handler struct{}

func NewHandler() Handler {
	return Handler{}
}

func Register(ctx context.Context, e common.EchoRouter, handler Handler) error {
	return nil
}
```

The `Handler` doesn't implement any endpoints yet, and the `Register` function register anything. Your job is to add the `RegisterCustomer` method and wire it up.

## Wiring the Handler

The wiring takes two lines. In your `Register` function, call `RegisterHandlers` with a strict handler wrapper:

```go
RegisterHandlers(e, NewStrictHandler(handler, nil))
```

{{tip}}

We generate the `openapi.yaml` spec for you in this training, so you don't need to learn OpenAPI syntax right now. In real projects, we recommend writing the spec by hand or using AI to help you generate it. If you're curious, see the [OpenAPI 3.0 Specification](https://swagger.io/specification/).

{{endtip}}

{{tip}}

Every new endpoint follows the same pattern: add it to the spec, regenerate, implement the new method on `StrictServerInterface`. The compiler will tell you what's missing.

{{endtip}}

## Exercise

Exercise path: ./project

Customers need an account to place orders. Let's implement the first endpoint to allow them to register.

The endpoint definition is already in `backend/orders/api/http/openapi.yaml`.
It's a POST request that accepts basic customer details like name, email, phone, and address, then returns a UUID.

1. Generate the OpenAPI code by running `task gen` from the project root. (If you haven't installed Taskfile, see the [installation guide](https://taskfile.dev/installation/).) You can also run `go generate ./...` in the project directory.

2. In `backend/orders/api/http/handler.go`, add a `RegisterCustomer` method on `Handler` to satisfy `StrictServerInterface`.

3. For now, return a `RegisterCustomer201JSONResponse` with a UUID from `uuid.New()` (`github.com/google/uuid`).

4. In the `Register` function in the same file, call `RegisterHandlers(e, NewStrictHandler(handler, nil))` to set up the routing.

{{hints}}

{{hint 1}}

Open `openapi.gen.go` and search for `StrictServerInterface`. The method signature you need to implement is right there.

{{endhint}}

{{hint 2}}

An example solution can look like this:

```go
func (h Handler) RegisterCustomer(ctx context.Context, request RegisterCustomerRequestObject) (RegisterCustomerResponseObject, error) {
    customerUUID := uuid.New()

    return RegisterCustomer201JSONResponse{
        CustomerUuid: customerUUID,
    }, nil
}

func Register(ctx context.Context, e common.EchoRouter, handler Handler) error {
    RegisterHandlers(e, NewStrictHandler(handler, nil))

    return nil
}
```

Don't forget the import: `"github.com/google/uuid"`.

{{endhint}}

{{endhints}}
