# Copilot COBOL Account App (Go)

An interactive, menu-driven account management console. It is a migration of the COBOL `MainProgram` / `Operations` / `DataProgram` programs to Go. The program reads menu choices from standard input, writes prompts and results to standard output, and keeps a single account balance in process memory.

This is a terminal program, not a network service. It does not open a port or serve HTTP.

## Migration status

| Item | Status |
|---|---|
| Modules migrated | 3 / 3 |
| Build | Passed |
| Target unit tests (`go test ./...`) | Passed |
| Behaviour compared with the original COBOL program | **Inconclusive** |
| Model self-assessed confidence | 93% (an estimate, not a measured result) |

Equivalence with the original program has not been demonstrated. See [Manual review required](#manual-review-required).

## Tech stack

- **Language:** Go. The module is `accountsystem` and `go.mod` declares `go 1.23.0`.
- **Libraries:** Go standard library (`context`, `os/signal`, `sync`, and others).
- **Declared dependency:** `github.com/spf13/viper` v1.21.0. This dependency is listed in `go.mod`, but neither entry point shown below imports it.
- **Storage:** In-memory only. There is no database.

## Prerequisites

- A Go toolchain matching the one used during migration (the `golang:1.27-alpine` image, Go 1.27). `go.mod` requires at least Go 1.23.0.
- Network access for `go mod download` on the first build.

No database, container runtime, or other services are required.

## Getting started

Run these steps from the root of a fresh clone.

### 1. Install dependencies and build

```sh
mkdir -p bin
go mod download
go build -o bin/app ./cmd/app
```

### 2. Environment setup

No environment setup is required to run the program. See [Environment variables](#environment-variables).

### 3. Database setup

None. The balance lives in memory and resets each time the process starts. No schema is created.

### 4. Run

```sh
./bin/app
```

The program shows a menu on the terminal and waits for input. Enter `4` to exit. The program then prints `Exiting the program. Goodbye!` and exits with status 0. Sending end-of-input (Ctrl-D) at the menu prompt has the same effect.

There is no port to connect to and no HTTP request to try. No sample sessions have been verified against the original program.

#### Alternative entry point

The repository contains a second entry point, `cmd/accountsystem`. It runs the same console, but it also handles Ctrl-C (SIGINT) and exits with status 130 when interrupted.

```sh
go run ./cmd/accountsystem
```

The migration run did not build or exercise this entry point. Only `./cmd/app` was built.

## Running tests

```sh
go test ./...
```

Test files:

- `internal/account/service_test.go`
- `internal/cli/menu_test.go`
- `internal/money/money_test.go`
- `internal/store/store_test.go`

## Environment variables

The migration sandbox defined the following variables. Neither entry point reads them, and `internal/store/store.go` states that no database is involved. You do not need to set them to run the program.

| Variable | Used by the code shown | Notes |
|---|---|---|
| `DATABASE_URL` | No | No database is used. The balance is held in memory. |
| `PORT` | No | The program is a stdin/stdout console and does not listen on a port. |
| `JWT_SECRET` | No | No authentication is implemented. |

Before relying on this table, confirm that nothing under `internal/` reads these variables, for example through `viper`.

## Architecture overview

```
cmd/
  app/main.go            Entry point built by the install step (bin/app)
  accountsystem/main.go  Same wiring, plus SIGINT handling (exit code 130)
internal/
  account/service.go     Account operations service (built on a BalanceStore)
  cli/menu.go            Interactive menu loop: Run(ctx, stdin, stdout, svc)
  money/money.go         money.Amount type, InitialBalance, range validation
  store/store.go         BalanceStore interface and in-memory implementation
```

**Wiring.** Both entry points follow the same sequence:

1. Create `store.NewMemoryStore(money.InitialBalance)`.
2. Wrap the store in `account.NewService`.
3. Hand the service to `Run` in `internal/cli/menu.go` along with stdin and stdout.

Errors are printed to stderr and produce exit status 1.

**`internal/store`.** This package replaces the COBOL `DataProgram` subprogram and its `WORKING-STORAGE` balance:

- `MemoryStore` is guarded by a mutex and implements `Read` and `Write`.
- `Write` rejects any value outside the `PIC 9(6)V99` range (0 to 999999.99) with `ErrOutOfRange` and leaves the stored balance unchanged.
- `store.Call(ctx, s, op, &balance)` reproduces the COBOL `CALL 'DataProgram' USING op, balance` interface. Only the first six characters of `op` are significant and trailing spaces are ignored, matching `PIC X(6)`. `"READ"` copies the stored balance out and `"WRITE"` stores it. Any other code does nothing and returns `nil`, as in the original.

**Mapping from COBOL.** `DataProgram` maps to `internal/store`. `Operations` and `MainProgram` appear to correspond to `internal/account` and `internal/cli`.

## Migration notes

The original COBOL program used no database (`cobol/none`), and neither does the Go version. The balance is still held only for the lifetime of the process, mirroring the COBOL run unit.

Changes from the original:

1. **EOF at the menu prompt exits cleanly.**
   - **Change:** End of input at the menu prompt is treated as choice 4. The program prints `Exiting the program. Goodbye!` and exits with status 0. The original GnuCOBOL program left `USER-CHOICE` unchanged on EOF and looped forever.
   - **Reason:** This was a deliberate deviation in the migration plan. The original's EOF behaviour is an infinite loop, so the Go version ends the session cleanly instead.

2. **Explicit range check on balance writes.**
   - **Change:** The `Write` method of `MemoryStore` in `internal/store/store.go` validates the amount and returns `ErrOutOfRange` for values outside 0 to 999999.99.
   - **Reason:** This is documented as a `MIGRATION_NOTE` in `internal/store/store.go`. In COBOL, the caller's field shared the same PIC, so an out-of-range value could never reach the store. Go callers can pass any value, so the store validates explicitly instead of truncating silently.

## Known limitations

- **Behavioural equivalence is unverified.** The comparison against the original COBOL program was inconclusive, and no input/output sessions have been confirmed to match.
- **No persistence.** The balance resets to `money.InitialBalance` on every start. This matches the original but may surprise users who expect saved state.
- **Two entry points.** `cmd/app` and `cmd/accountsystem` are near-duplicates. Only `cmd/app` was built during migration.
- **Unused dependency and configuration.** `viper` is declared in `go.mod`, and `DATABASE_URL`, `PORT` and `JWT_SECRET` were set in the sandbox, but the entry code shown uses none of them.

No components were left untranslated and none were marked unmigratable.

## Manual review required

Review these items before treating this migration as complete:

- **Low-confidence modules from the original COBOL `Operations` and `MainProgram` programs.** Their Go counterparts appear to be:
  - `internal/account/service.go` (from `Operations`)
  - `internal/cli/menu.go` (from `MainProgram`)

  Check the following against the original programs:
  - menu text and option numbering
  - credit and debit arithmetic and rounding
  - insufficient-funds handling
  - output formatting of amounts
  - handling of invalid menu input
- **`internal/money/money.go`.** Confirm that `InitialBalance` and the `PIC 9(6)V99` validity rules match the original data definitions.
- **EOF behaviour in `internal/cli/menu.go`.** Confirm that the intentional deviation described in the migration notes is implemented as stated.
- **Entry points.** Decide whether to keep both `cmd/app/main.go` and `cmd/accountsystem/main.go` or consolidate them.
- **`go.mod`.** Confirm whether `github.com/spf13/viper` is needed. Remove it if not.
- **Side-by-side behaviour testing.** Run the same input scripts through the original COBOL program and through `./bin/app`, and compare the output. This verification is still missing.