# Copilot COBOL Account App (Go)

This project is a Go port of the Copilot COBOL Account App, a small account-management application. The original was written in COBOL with no framework. All 3 original modules were translated into Go using the standard toolchain.

> **Status:** The project builds, and the target unit tests (`go test ./...`) pass. Whether it **behaves like the original COBOL program has not been established**. The behavioral comparison was inconclusive, and no request or interaction has been confirmed to match the original. The 93% figure in the migration report is the migration model's self-assessed confidence. It is not a measured result.

## Tech stack

- **Language:** Go, standard library and toolchain (`go build`, `go test`)
- **Module definition:** `go.mod`
- **Toolchain used during migration:** Go 1.27 (the `golang:1.27-alpine` image)
- **Database:** a PostgreSQL instance was available during migration, and `DATABASE_URL` is listed as required (see [Database](#database))

## Prerequisites

- Go 1.27
- A reachable database for `DATABASE_URL`. During migration this was a PostgreSQL instance.

## Getting started

Run these steps in order from the root of a fresh clone.

### 1. Install dependencies and build

```sh
mkdir -p bin
go mod download
go build -o bin/app ./cmd/app
```

This builds the `cmd/app` entry point into `bin/app`.

### 2. Tidy module dependencies

These commands were run during migration after install and before running. The migration ran `go mod tidy` twice. The second run is idempotent, but both are listed here to mirror what succeeded.

```sh
go mod tidy
go mod tidy
```

### 3. Configure the environment

```sh
export DATABASE_URL="postgres://USER:PASSWORD@HOST:5432/DBNAME?sslmode=disable"
export PORT=8080
export JWT_SECRET="replace-with-a-long-random-secret"
```

The values above are placeholders. See [Environment variables](#environment-variables).

### 4. Database setup

No database setup commands (schema creation, migrations, seed data) were recorded during migration. If the application needs a schema, you must create it manually. See [Manual review required](#manual-review-required).

### 5. Run

No run command was recorded during migration. The build step produces `bin/app`, and starting that binary is the expected way to run the app:

```sh
./bin/app
```

This step has not been verified. The migration report states the app listens on **port 8080**, reachable at `http://localhost:8080`. However, the code also contains an interactive CLI menu (`internal/cli/menu.go`), so the actual mode of operation needs confirmation.

**First request to try:** none. No request has been verified to produce the same result as the original application.

## Database

| | Database |
|---|---|
| Original (COBOL) | Not recorded in the migration output |
| Migrated (Go) | `DATABASE_URL` is required. A PostgreSQL instance was provided during migration. |

The migration output does not record a database change, its reason, or whether `internal/store/store.go` actually connects to PostgreSQL.

## Running tests

```sh
go test ./...
```

This passed during migration. The only test file present is `internal/money/money_test.go`, so test coverage is limited to the `money` package.

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | Connection string for the application database (PostgreSQL during migration). |
| `PORT` | Yes | Port the application listens on. The migration report states `8080`. |
| `JWT_SECRET` | Yes | Secret used for JWT signing/verification. |

The migration flagged all three variables as needed. How each one is consumed in code has not been documented.

## Architecture overview

```
cmd/
  accountsystem/main.go   # entry point (not built by the recorded install step)
  app/main.go             # entry point built into bin/app
internal/
  account/service.go      # account operations (business logic)
  cli/menu.go             # interactive menu / user interaction
  money/money.go          # monetary value handling
  money/money_test.go     # unit tests for money
  store/store.go          # persistence / data storage
go.mod
```

The descriptions below are inferred from package and file names:

- **`cmd/app`** and **`cmd/accountsystem`** are two separate entry points. Only `cmd/app` is built by the recorded install step. The role of `cmd/accountsystem` relative to `cmd/app` has not been documented.
- **`internal/account`** holds the account service, which most likely corresponds to the original operations logic.
- **`internal/cli`** handles the menu-driven interaction carried over from the original COBOL program flow.
- **`internal/money`** is an isolated package for money arithmetic. It is the only package with unit tests.
- **`internal/store`** is the storage layer for account data.

## Migration notes

- **Source → target:** COBOL (no framework) → Go (standard toolchain).
- **Modules migrated:** 3 of 3.
- **Recorded changes from the original:** none. The migration did not record any intentional behavioral or structural changes, or reasons for them. The differences below are visible from the project structure but were not documented:
  - The app requires `DATABASE_URL`, `PORT`, and `JWT_SECRET`, and is reported to listen on port 8080.
  - It has two entry points under `cmd/`.

  Treat these as unexplained until reviewed.
- **Untranslated source files:** none.

## Known limitations

- **Behavioral parity is unconfirmed.** The comparison against the original COBOL program was inconclusive, and no requests or interactions are confirmed to match.
- **No run command was recorded.** The `./bin/app` step above is the expected invocation, but it was not exercised during migration.
- **No database setup was recorded.** Schema and seed requirements are unknown.
- **Test coverage is minimal.** Only `internal/money` has tests. `internal/account`, `internal/cli`, and `internal/store` have none.
- **The mode of operation is unclear.** The report says the app listens on 8080, but the code includes a CLI menu.

## Manual review required

**Low-confidence modules.** The Go code translated from these original modules needs careful line-by-line comparison against the COBOL source:

- The original operations module: most likely maps to `internal/account/service.go` and `internal/money/money.go`.
- The original main module: most likely maps to `cmd/app/main.go`, `cmd/accountsystem/main.go`, and `internal/cli/menu.go`.

**Additional files to verify:**

- `internal/store/store.go`: confirm how data is persisted, whether it uses `DATABASE_URL`, and what schema it expects.
- `cmd/app/main.go` and `cmd/accountsystem/main.go`: determine which is the intended entry point, whether it serves HTTP on `PORT` or runs an interactive menu, and where `JWT_SECRET` is used.
- `internal/money/money.go`: confirm that rounding and precision match COBOL fixed-point (`PIC`) semantics for balances, credits, and debits.
- `go.mod`: confirm the Go version and dependencies after the `go mod tidy` runs.