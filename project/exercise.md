# API Client

Testing HTTP endpoints is a lot of boilerplate. You need to build URL strings, marshal and unmarshal JSON, and check status codes.
Every new endpoint adds more code and another place to get the contract wrong.

But there's a better way. **The same [OpenAPI](https://academy.threedots.tech/knowledge/openapi) spec that generates your server handlers can also generate a typed Go client that handles all of it.**

### Client Generation

You've been using [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to generate server code from `backend/orders/api/http/openapi.yaml`. The tool works in the other direction too. With a different config, it generates a typed HTTP client from the same spec.

We've added a `generate.go` file in `backend/orders/api/http/client/`.

```go
package client

//go:generate go tool oapi-codegen --config=oapi-codegen.yaml ../openapi.yaml
```

The [`//go:generate`](https://go.dev/blog/generate) directive tells Go tooling to run oapi-codegen with a client-specific config. When you run `task gen` (or `go generate ./...`), Go scans for these directives and executes them.

The config tells oapi-codegen what to generate:

```yaml
package: client
output: client.gen.go
generate:
  models: true
  client: true
output-options:
  response-type-suffix: ClientResponse
```

Two fields matter here. `generate.models: true` creates Go structs for request and response schemas. `generate.client: true` creates the HTTP client with typed methods.

Together, they produce a `client.gen.go` file with everything you need to call the API.

### What You Get

After generation, the client package contains a `RegisterCustomer` method with typed request and response structs. The response wrapper gives you pre-parsed JSON for each status code:

```go
type RegisterCustomerClientResponse struct {
    Body         []byte
    HTTPResponse *http.Response
    JSON201      *RegisterCustomerResponse  // success
    JSON400      *BadRequest                // validation error
    JSON409      *ErrorResponse             // conflict
}
```

Look at the types used in the generated package. `CustomerUUID` is `app.CustomerUUID`, and `CountryCode` is `shared.CountryCode`.
**The `x-go-type` mappings you added to the OpenAPI spec carry over to the generated client.**
The client uses the exact same types as the server.

In the next module, you'll use this client to write [component tests](https://academy.threedots.tech/knowledge/component-test).
Instead of constructing HTTP requests by hand, you'll call typed methods with auto-completion and compile-time checks.

{{tip}}

The `response-type-suffix: ClientResponse` config avoids a naming collision.
 The server already generates a `RegisterCustomerResponse` struct (the JSON payload).
 The client needs its own response wrapper that includes the HTTP status code and parsed body.
 Without the suffix, both would be named `RegisterCustomerResponse`.
 With it, the client's wrapper becomes `RegisterCustomerClientResponse`.

{{endtip}}

{{tip}}

We wrote about oapi-codegen and other recommended Go libraries in [The Go libraries that never failed us](https://threedots.tech/post/list-of-recommended-libraries/). For more on testing with generated clients, see [Microservices test architecture](https://threedots.tech/post/microservices-test-architecture/).

{{endtip}}

## Exercise

Exercise path: ./project

Run `task gen` (or `go generate ./...`) to generate the HTTP client code.

After running the command, check that `backend/orders/api/http/client/client.gen.go` was created.
It should contain a `Client` struct with a `RegisterCustomer` method and typed request/response structs using your domain types.
