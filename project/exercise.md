# Error Handling

Right now, every error in your application reaches the user as an HTTP 500 with a generic message.
A database timeout looks the same as invalid input.
All the frontend app can do is to show a generic "Something went wrong" message, which isn't very helpful for our users.

We'll fix that by adding structured errors in the [application layer](https://academy.threedots.tech/knowledge/application-service), so `RegisterCustomer` can tell the API layer what exactly went wrong.

## Transport-Agnostic Errors

Currently, the `RegisterCustomer` app service method is called only by the HTTP handler.
But in the future, it could be called by a [gRPC](https://academy.threedots.tech/knowledge/grpc) handler, a [Pub/Sub](https://academy.threedots.tech/knowledge/message-broker) message handler, or a CLI tool.
If the service returns `echo.NewHTTPError(400, "invalid input")`, what happens when a message handler calls the same method?
The handler receives an HTTP error, but it doesn't have an HTTP request, a response, or a status code to set.
Logging "HTTP 400 Bad Request" when processing a message is confusing and misleading.

**The application layer should express *what went wrong*, not *what to tell the client*.**
It should return `invalid input`, `not found`, or `unauthorized` error.
The API layer translates that into HTTP 400, gRPC `INVALID_ARGUMENT`, or whatever the protocol needs.

## Error Slugs

So how do we identify errors without coupling to HTTP? We use **error slugs**: human-readable string identifiers like `"empty-name"` or `"customer_not_found"`.

Compare `error_code: 40012` with `error_slug: "empty-name"`. Which one can you easily grep in logs? Which one can a frontend developer map to a user message without checking the docs?

Slugs give you two things: they're **human-readable for debugging and logging**, and they're a **stable API contract the frontend (and other clients) relies on**.

If the frontend uses `"empty-name"` to show "Please enter your name," changing the slug breaks the frontend. We'll test slugs as part of the API contract in a later exercise.

## The `common` Error Package

We implemented a `common` error package that standardizes how the project handles errors. Take a look at the `Error` type in:

{{codeFile "backend/common/errors.go"}}

```go
type Error struct {
	HttpErrorCode int

	PublicError string
	ErrorSlug   string

	InternalError error
	Details       []ErrorDetails
}
```

| Field           | Type             | Purpose                                                        |
|-----------------|------------------|----------------------------------------------------------------|
| `HttpErrorCode` | `int`            | HTTP status code, set by constructor, read by handler          |
| `PublicError`   | `string`         | Human-readable message sent to the client                      |
| `ErrorSlug`     | `string`         | Machine-readable slug                                          |
| `InternalError` | `error`          | Internal details for logging. **Never exposed to the client.** |
| `Details`       | `[]ErrorDetails` | Per-field validation details                                   |

The `Error` type also has two builder methods:

* `WithDetails([]ErrorDetails)` appends field-level details.
* `WithInternalError(error)` attaches internal context for logging.

Both return a new `Error` value without changing the original.

Each constructor maps to a specific HTTP status code:

| Constructor            | HTTP Code        | When to use                           |
|------------------------|------------------|---------------------------------------|
| `NewInvalidInputError` | 400 Bad Request  | Validation failures, malformed input  |
| `NewUnauthorizedError` | 401 Unauthorized | Authentication/authorization failures |
| `NewNotFoundError`     | 404 Not Found    | Entity does not exist                 |
| `NewExpiredError`      | 410 Gone         | Resource has expired or been removed  |

All four share the same signature: `(slug, publicErrorFormat string, a ...any)` where the format string supports `fmt.Sprintf` formatting.
For most applications, these four constructors cover everything you need.

```go
return common.NewNotFoundError(
    "customer_not_found",
    "Customer %s not found",
    customerUUID,
)
```

{{tip}}

`WithInternalError` is useful when wrapping infrastructure errors. For example, if a database query fails, you can attach the original error for logging without exposing it to the client.
You'll use this pattern in later exercises when working with repositories.

{{endtip}}

## Error Details

When a user submits a registration form with three mistakes, returning only the first error can be frustrating.
The user needs to fix one field, submit, see the next error, fix it, and submit again.
A better UX would be to **collect all validation errors and return them together**, so the frontend can show all of them at once.

The `ErrorDetails` struct includes per-field information: `EntityType`, `EntityID`, `ErrorSlug`, and `Message`.
For validation, the `EntityType` and `ErrorSlug` fields are what matter most. The rest is there when the frontend needs it.

When `RegisterCustomer` returns validation errors, the HTTP response looks like this:

```json
{
  "message": "Invalid customer data",
  "slug": "invalid_customer_data",
  "details": [
    {
      "entity_type": "customer",
      "entity_id": "550e8400-...",
      "error_slug": "empty-name",
      "message": "Name cannot be empty"
    },
    {
      "entity_type": "customer",
      "entity_id": "550e8400-...",
      "error_slug": "empty-phone-number",
      "message": "Phone number cannot be empty"
    }
  ]
}
```

The frontend can use the entity type and id to highlight specific form fields and `error_slug` to show the user what to fix.

## Setting Up the Error Handler

For this JSON response to work, Echo needs to know how to convert `common.Error` into HTTP responses. One line adds the error handler for all routes:

```go
e.HTTPErrorHandler = common.EchoErrorHandler
```

`EchoErrorHandler` is called whenever a handler returns a non-nil error.
It checks if the error is a `common.Error` (using `errors.As`), extracts the slug and HTTP code, and returns a JSON response.

What happens with errors that aren't a `common.Error`?
They become a generic HTTP 500 with `{"message": "Internal Server Error", "slug": "internal_server_error"}`.
**Internal details are never leaked.**

See the [Echo Custom HTTP Error Handler](https://echo.labstack.com/docs/error-handling#custom-http-error-handler) docs for more on how this works.

{{tip}}

We wrote more about the motivation for transport-agnostic errors in [How to implement Clean Architecture in Go](https://threedots.tech/post/introducing-clean-architecture/). The article shows how the same error pattern works across both HTTP and gRPC handlers.

{{endtip}}

## Exercise

Exercise path: ./project

1. **Change the Echo error handler in `backend/common/http/echo.go`** after the Echo instance is created:

    ```go
    e.HTTPErrorHandler = common.EchoErrorHandler`
    ```

2. **Add validation to `RegisterCustomer`** in `backend/orders/app/customer.go`. The service should validate each field of the `Customer` struct:

    - `CustomerUUID` should not be zero (use `IsZero()`)
    - `Name` should not be empty (use `strings.TrimSpace` to catch whitespace-only input)
    - `Email` should not be empty (same trimming)
    - `Address` should not be zero (use `Address.IsZero()`)
    - `PhoneNumber` should not be empty (same trimming)

**Collect all validation errors into a `[]common.ErrorDetails` slice before returning.** Do not return on the first failure.
Each failed check should append an `ErrorDetails` with `EntityType: "customer"`, the customer's UUID as `EntityID`, and `ErrorSlug` set to `"empty-name"` or `"empty-phone-number"`.

If any validations fail, return `common.NewInvalidInputError("invalid_customer_data", "Invalid customer data").WithDetails(errDetails)`.

The platform will verify that your customer registration endpoint returns HTTP 201 with a valid customer UUID and returns HTTP 400 with the correct error slugs (`"empty-name"`, `"empty-phone-number"`) when fields are invalid.

{{tip}}

The HTTP handler auto-generates the UUID and validates the address via `shared.NewAddress()` before calling `RegisterCustomer`.
So the UUID, email, and address validations are defensive checks that protect against direct service calls (from tests, message handlers, or other internal callers).
For the HTTP flow, only `Name` and `PhoneNumber` validations are reachable. That's still good practice.

{{endtip}}

{{hints}}

{{hint 1}}

Here's the pattern for validating two fields. Extend it for the remaining three (Email, Address, PhoneNumber):

```go
errDetails := []common.ErrorDetails{}

if customer.CustomerUUID.IsZero() {
    errDetails = append(errDetails, common.ErrorDetails{
        EntityType: "customer",
        EntityID:   "",
        ErrorSlug:  "empty-uuid",
        Message:    "UUID cannot be empty",
    })
}
if strings.TrimSpace(customer.Name) == "" {
    errDetails = append(errDetails, common.ErrorDetails{
        EntityType: "customer",
        EntityID:   customer.CustomerUUID.String(),
        ErrorSlug:  "empty-name",
        Message:    "Name cannot be empty",
    })
}

// TODO validate Email, Address, PhoneNumber

if len(errDetails) > 0 {
    return common.NewInvalidInputError(
        "invalid_customer_data",
        "Invalid customer data",
    ).WithDetails(errDetails)
}
```

{{endhint}}

{{endhints}}
