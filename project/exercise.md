# Currency Enum

**Where do you keep a type that multiple modules need?**

Consider a `Currency` type.
If it lives in `orders`, and another module needs it, then that module depends on `orders`.
We want to avoid such coupling to keep modules independent.

There are two ways to go about this:

1. Keep a separate type for both modules.
2. Put the type in a shared package that both modules import.

It's a delicate balance, so you need to consider the specific use case.
For our project, we'll keep `Currency` in the `common/shared` package, similarly to `CountryCode`.

`Currency` will also use the same `Enum[T]` pattern.

## Shared vs. Module-Specific Types

Currency works as a shared type because it's rather generic: a list of valid currency codes with no extra business logic.
There is no validation beyond "is this a known code?", and no behavior specific to one module.

**Watch out for business logic in `common` packages.
It's one of the worst sources of coupling.**
Every module ends up depending on it, and changing shared behavior affects the entire project.

Shared packages aren't forbidden, but keeping the logic there is almost always a bad idea.
Stick to plain data types like `Currency` and `CountryCode`.

If a module needed specialized currency logic (say, a pricing module supporting cryptocurrency codes that other modules don't recognize),
adding those codes to the shared `Currency` would force every module to handle values it doesn't care about.
In that case, the pricing module should define its own `Currency` type.

**Share a type when it's used the same way everywhere.
Keep it local when it has business logic or means different things in different modules.**

{{conversation "From a Past Code Review"}}

{{message "milosz"}}

I'm a bit worried about putting `Currency` in `common/shared`. We've seen shared packages turn into dumping grounds. How do we keep it under control?

{{endmessage}}

{{message "robert"}}

The rule I follow: a shared type should be a plain data type with no business logic. `Currency` is just a list of valid codes. If a module ever needs special currency behavior (like supporting crypto codes that other modules don't recognize), that module should define its own type.

{{endmessage}}

{{message "milosz" "robert:+1"}}

So the test is: does every module use this type the same way? If yes, share it. If any module needs different behavior, keep it local.

{{endmessage}}

{{endconversation}}

## SharedTypes and cmp.Diff

Before we get to the implementation, let's talk about why `Currency` needs to be registered in the `SharedTypes` slice.

The `Enum[T]` generic type stores its value in an unexported field:

```go
type Enum[T Enumerable] struct {
    value string  // unexported
}
```

[`cmp.Diff`](https://pkg.go.dev/github.com/google/go-cmp/cmp), which we use in repository tests to compare structs, panics when it encounters unexported fields. It can't read them through reflection. Without special handling, comparing any struct that contains a `Currency` or `CountryCode` fails with:

```text
cannot handle unexported field at {Address.CountryCode.Enum}.value
```

[`cmpopts.EquateComparable`](https://pkg.go.dev/github.com/google/go-cmp/cmp/cmpopts#EquateComparable) fixes this.
It tells `cmp.Diff` to use Go's built-in `==` operator for specific types instead of inspecting their fields.
The `SharedTypes` slice has all the types that need this:

```go
cmpopts.EquateComparable(shared.SharedTypes...)
```

Once you add `Currency{}` to `SharedTypes`, every test that uses this option picks it up automatically.

## Exercise

Exercise path: ./project

Create a `Currency` enum in `backend/common/shared/currency.go`, following the `CountryCode` pattern:

1. Create a `CurrencyType` string type with a `Values()` method returning:
    ```go
    []string{"USD", "EUR", "GBP", "JPY", "PLN"}
    ```
2. Create a `Currency` struct embedding `Enum[CurrencyType]`
3. Add a `Code() string` method on `Currency` returning the string value
4. Add a `MustNewCurrency(value string) Currency` constructor (works like `MustNewCountryCode`)
5. Add `Currency{}` to the `SharedTypes` slice in `backend/common/shared/types.go`

{{hints}}

{{hint 1}}

For the constructor, use `UnmarshalText` to validate the input, just like `MustNewCountryCode` does.
Look at the `CountryCode` implementation in `backend/common/shared/country_code.go` for the exact pattern.

{{endhint}}

{{hint 2}}

Here's one way to implement it:

```go
type Currency struct {
	Enum[CurrencyType]
}

func (c Currency) Code() string {
	return c.value
}

type CurrencyType string

func (c CurrencyType) Values() []string {
	return []string{"USD", "EUR", "GBP", "JPY", "PLN"}
}

func MustNewCurrency(value string) Currency {
	c := Currency{}
	err := c.UnmarshalText([]byte(value))
	if err != nil {
		panic(fmt.Errorf("error unmarshalling currency value: %s", value))
	}
	return c
}
```

{{endhint}}

{{endhints}}
