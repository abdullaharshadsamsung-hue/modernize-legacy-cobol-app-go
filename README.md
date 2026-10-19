# Copilot COBOL Account App (Go)

This is a Go port of the Copilot COBOL Account App, an account application originally written in COBOL. All three original modules have been translated to Go. Some parts of the migration are still unverified. See [Known limitations](#known-limitations) and [Manual review required](#manual-review-required) before you rely on this code.

**Migration status**

| Check | Result |
|---|---|
| Build | Passed |
| Target unit tests (`go test ./...`) | Passed |
| Behavior compared with the original COBOL app | **Inconclusive** |
| Modules migrated | 3 / 3 |
| Model self-assessed confidence | 93% (not a measured result) |

## Tech stack

- **Language:** Go, standard library layout (`cmd/`, `internal/`)
- **Module definition:** `go.mod`
- **Database:** reached via `DATABASE_URL`. The migration environment provided a PostgreSQL service.

## Prerequisites

- **Go 1.27.** The migration was built and tested with the `golang:1.27-alpine` toolchain.
- **A database reachable through `DATABASE_URL`.** During migration, a PostgreSQL instance was available at host `postgres`.
- Port **8080** free on your machine, or a different port set with `PORT`.

## Getting started

Run these steps in order from the root of a fresh clone.

### 1. Install dependencies and build

```sh
mkdir -p bin
go mod download
go build -o bin/app ./cmd/app
```

### 2. Set environment variables

```sh
export DATABASE_URL="<your database connection URL>"
export PORT=8080
export JWT_SECRET="<a secret value>"
```

See [Environment variables](#environment-variables) for details.

### 3. Set up the database

No database setup or migration commands were recorded during the migration. Before you start the app, make sure `DATABASE_URL` points to a reachable database. Then check `internal/store/store.go` to see what schema or tables the app expects.

### 4. Run the app

The migration did not record an explicit start command. The build step produces the binary `bin/app`, and the app listens on port **8080**:

```sh
./bin/app
```

You can reach the app at `http://localhost:8080`, or on the port set in `PORT`.

**First request to try:** none yet. No requests were verified against the migrated app during migration, so there is no confirmed example request. The project also contains a CLI menu (`internal/cli/menu.go`). Check `cmd/app/main.go` to see whether the app starts an interactive menu, an HTTP listener, or both.

## Running tests

```sh
go test ./...
```

These tests passed during migration. They test the Go code itself. They do not show that the Go app behaves the same as the original COBOL app.

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | Connection URL for the database used by the store layer (`internal/store/store.go`). |
| `PORT` | Yes | Port the app listens on. The migration ran the app on `8080`. |
| `JWT_SECRET` | Yes | Secret used for JWT handling. Use a strong, private value outside local development. |

Configuration is loaded in `internal/config/config.go`.

## Architecture overview

```
cmd/app/main.go              Entry point: wires config, store, service and CLI together
internal/config/config.go    Reads configuration (DATABASE_URL, PORT, JWT_SECRET)
internal/account/service.go  Account business logic (ported from the COBOL operations)
internal/cli/menu.go         Menu-driven user interaction (ported from the COBOL main program)
internal/store/store.go      Persistence layer backed by DATABASE_URL
go.mod                       Go module definition and dependencies
```

The design keeps the original COBOL program boundaries as separate packages:

- The entry and menu flow lives in `cmd/app` and `internal/cli`.
- Account operations live in `internal/account`.
- Data storage lives in `internal/store`.

Everything except the entry point lives under `internal/`, so it cannot be imported from outside this module.

## Migration notes

- **Source:** COBOL, with no framework.
- **Target:** Go, standard project layout.
- **Recorded changes from the original:** none. The migration did not record any intentional behavior or design changes, with reasons.
- **Untranslated source files:** none.
- **Unmigrable components:** none.

## Known limitations

- **Behavioral equivalence is unconfirmed.** The comparison between the Go app and the original COBOL app was inconclusive. No requests or interactions have been verified to produce the same results as the original.
- **No verified first request.** There is no confirmed working example request or session.
- **No recorded run command or database setup.** The start command and any schema setup were not captured during migration. Steps 3 and 4 above are based on the build output and the required environment variables, not on a verified run.
- **Storage may have changed.** The original COBOL app's storage mechanism was not recorded. It is unclear whether the move to a `DATABASE_URL`-backed store changes how data is persisted compared with the original.

## Manual review required

The migration flagged these original modules as **low confidence**. Review their Go counterparts carefully against the COBOL sources:

| Original module | Go code to review | What to check |
|---|---|---|
| COBOL operations module | `internal/account/service.go`, `internal/store/store.go` | Arithmetic and balance handling (COBOL fixed-point decimal vs. Go numeric types), rounding, overflow, and validation rules. |
| COBOL main program | `cmd/app/main.go`, `internal/cli/menu.go` | Menu options, input parsing, prompts and output formatting, loop and exit behavior. |

Also check these:

- `internal/config/config.go`: confirm how a missing or invalid `DATABASE_URL`, `PORT`, or `JWT_SECRET` is handled.
- Run the original COBOL app and the Go app side by side with the same inputs, and compare the outputs. This is the check that came back inconclusive during migration.
