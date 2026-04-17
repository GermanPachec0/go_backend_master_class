{{tip}}
In some exercises, we'll push code changes to your workspace. You can always see what changed by clicking the link above.
{{endtip}}

# Service Scaffolding

{{message "robert"}}

Welcome aboard! 👋

We've built the scaffolding for what will become our food delivery platform: a modular monolith with an HTTP server and a way to talk between modules.

It looks like overkill for one empty service, but there's something you should know. By the end of this training, this "premature" structure will hold over 13,000 lines of production-grade code.

The project you'll build handles restaurant onboarding, customers placing orders, and courier delivery flow.

Right now it's an empty shell, but that's about to change.

{{endmessage}}

## Working Together

In this training, you're working with us on a real project. You'll see how a well-structured codebase looks when people care about maintainability from day one.

We've heard every excuse: "we don't have time for proper structure," "it's overengineering," "it only works for greenfield projects." Those concerns fall apart once you've seen it done right. This training gives you that experience.

You'll implement the parts of the project that you'll benefit most from. We've prepared the rest, like the scaffolding in this exercise. You can always change it to your liking later, but keep in mind that modifying shared code means extra work when merging real pull requests in the future.

You don't need to understand every file right now. We'll guide you through them as they become relevant.

## The Project Structure

All the code lives in the `backend` directory. This is a *monorepo*, so we could also keep the frontend code next to it.

Inside, you'll see three packages:

* `cmd`: the entry point of the application. It contains the `main` function and the code to start the server. We won't touch it much.
* `common`: infrastructure code common to all modules, like HTTP setup, [middleware](https://academy.threedots.tech/knowledge/middleware), and logging.
* `orders`: the first module you'll work on.

There's also the `backend/svc.go` file with the service initialization logic. You don't need to worry about it, unless you want to understand how it works.

## Why a Modular Monolith

We're building a **modular monolith**. We believe in the [monolith-first](https://martinfowler.com/bliki/MonolithFirst.html) approach and use it in most of our projects. The idea is to start with a single deployable unit where modules communicate within the process, rather than splitting into microservices from day one.

Architecturally, a well-structured modular monolith and microservices look the same: isolated modules with clear boundaries. The only difference is the network boundary and independent deployment.

In early phases, domain boundaries change a lot as you learn more about the problem. Splitting into microservices too early locks you into boundaries you don't fully understand yet, and moving code between services is far more expensive than moving it between modules.

There are fewer resources about modular monoliths than microservices, but the approach is battle-tested.

{{tip}}

We discussed monolith-first vs. microservices in our podcast episode [The Distributed Monolith Trap](https://threedots.tech/episode/the-distributed-monolith-trap/). If you want to go deeper, see [Microservices or Monolith: it's a detail](https://threedots.tech/post/microservices-or-monolith-its-detail/).

{{endtip}}

## The Module Interface

The core concept of the scaffolding is the `Module` interface in `backend/common/module/module.go`. Every module in the project implements four methods: `Name`, `Init`, `RegisterHttp`, and `RegisterContracts`.

The initialization sequence in `backend/svc.go` runs them in a specific order:

1. **`Init`** for each module (creates handlers, services, repositories)
2. **`RegisterContracts`** for each module (registers module contracts)
3. **`Verify`** on the contracts registry (checks all expected implementations are registered)
4. **`RegisterHttp`** for each module (sets up HTTP routes)

**The order matters.** Module contracts must be fully registered before HTTP routes are set up, because HTTP handlers may call other modules via module contracts.

The `Verify` call is a fail-fast safety net. If any module forgets to register its contract implementation, the service won't start.

## Building Blocks

We chose [Echo](https://echo.labstack.com/) for HTTP routing. Unlike `chi` or the standard library's `ServeMux`, Echo handlers return `error`. This lets you handle errors in one place (`HandleError` in `backend/common/http/echo.go`) instead of repeating it in every handler.

The `common` package holds shared infrastructure: HTTP setup, middleware, the module contracts registry, and the `Module` interface.

The `common` package is sometimes considered an anti-pattern, but it depends on what you keep there. In our case, it's infrastructure code that any module needs. No business logic.

We use `pgxpool.Pool` for database access because it manages a pool of connections, which is essential once your service handles concurrent requests.

The project also keeps a `*sql.DB` for compatibility with libraries that require the standard `database/sql` interface.

Modules communicate through typed Go interfaces called **module contracts**. This enforces module boundaries within the process, without network overhead. If one module needs something from another, it calls a method on a contract interface, not a raw function.

With microservices, the network is the boundary, and you're only able to call the service's public API methods.
In a modular monolith, we enforce it with code structure.

The `PingOrders` method you'll see is a placeholder. By the time you finish the module contracts section, modules will communicate over this layer. For now, it exists so the contracts wiring can be verified at startup.

## Exercise

Exercise path: ./project

There's nothing to implement here. Take a look at the code we prepared and continue to the next exercise.

In the next exercise, we'll look at structured logging and request [tracing](https://academy.threedots.tech/knowledge/tracing).
