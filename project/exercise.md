# Initial Project

The difference between a good and bad local dev environment is measured in hours per week. Slow feedback loops, manual restarts, and flaky setups waste more time than most people realize. We've designed this project so you can focus on writing Go, not fighting infrastructure.

Let's walk through the setup.

## Taskfile

We use [Task](https://taskfile.dev/) for task automation. If you've worked with Makefiles before, Task is a similar idea but with YAML syntax, cross-platform support, and built-in task dependencies and variables.

{{tip}}

To install Task, see the [installation guide](https://taskfile.dev/installation/). It's not mandatory. You can always run the underlying commands directly (like `docker compose up` or `go test ./...`), but Task wraps them into short, convenient commands.

{{endtip}}

Open `Taskfile.yml` and take a look. The tasks are grouped by purpose:

- **Docker lifecycle**: `up`, `up-clean`, `down`, `down-volumes`
- **Testing**: `test` (all tests), `test-unit` (unit tests only — later you'll also get `test-integration` and `test-component`)
- **Code quality**: `fmt`, `lint`, `tidy`
- **Utilities**: `gen` (code generation), `pgcli` (connect to PostgreSQL)

You won't need most of these right away. They'll make sense as the project grows. The Taskfile you see now is close to what you'll have at the end of the training. It doesn't change much.

## Running Locally with Docker Compose

**You can complete the entire training without Docker installed.** The `tdl` CLI handles everything: it compiles your code, runs it on our servers, and validates your solution.

{{tip}}

If you don't have Docker installed on your machine, you can download [Docker Desktop](https://www.docker.com/products/docker-desktop/).

{{endtip}}

That said, if you want to run the project locally, it's one command:

```bash
task up
```

Or equivalently, `docker compose up`. This starts two services: your Go backend on port 8080 and a PostgreSQL database.

Use `task up-clean` to stop and restart from a clean state. When using `tdl tr run`, the project is running on our servers, always in a clean state, so you don't need to worry about leftover data there.

The Docker Compose setup includes auto-reload with [reflex](https://github.com/cespare/reflex). When you save a `.go`, `.mod`, or `.sql` file, reflex automatically rebuilds and restarts the application. The configuration is in `backend/reflex.conf`.

{{tip}}

We wrote about setting up Docker development environment with live reloading in [Go Docker Dev Environment with Go Modules and Live Code Reloading](https://threedots.tech/post/go-docker-dev-environment-with-go-modules-and-live-code-reloading/).

{{endtip}}

## Exercise

Exercise path: ./project

This one is on us. The project compiles as-is.

If you want start the project locally, run `task up` or `docker compose up` and see the logs. You should see a `Hello, World!` printed out. If you change the `main.go`, the server should reload automatically.

Submit your solution with `tdl tr run` and move to the next exercise, where we'll set up the project scaffolding.
