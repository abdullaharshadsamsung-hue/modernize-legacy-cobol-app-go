# Copilot COBOL Account App (Go)

This is an interactive, menu-driven console program that manages a single account balance. It was migrated from three COBOL programs (`MainProgram`, `Operations` and `DataProgram`) to Go.

The program reads menu choices from standard input and writes results to standard output. The balance is held in process memory only. It starts at a fixed initial balance (`money.InitialBalance`) and resets each time the process starts, the same way the COBOL run unit did. No database is used.

> **Verification status**
>
> | Check | Result |
> |---|---|
> | Build | passed |
> | Unit tests (`go test ./...`) | passed |
> | Behaviour compared with the original COBOL program | **inconclusive** |
>
> The migration tool reported 93% confidence. That figure is the model's own estimate, not a measured result. Behavioural equivalence with the COBOL original has **not** been confirmed. See [Manual review required](#manual-review-required).

## Tech stack

- **Language:** Go. The module `accountsystem` declares `go 1.22` as its minimum version.
- **Libraries:** Go standard library only (`context`, `os/signal`, `sync`, etc.).
- **Storage:** in-memory, through a mutex-protected `store.MemoryStore`.
- **Interface:** interactive terminal program (stdin/stdout). It is not an HTTP server.

## Prerequisites

- Go toolchain. The migration was built and tested with the toolchain from the `golang:1.27-alpine` image (Go 1.27). Any version ≥ 1.22 satisfies `go.mod`.
- No database or other external service is required.

## Getting started

Run these steps from the root of a fresh clone, in this order.

### 1. Download dependencies and build

```sh
mkdir -p bin && go mod download && go build -o bin/app ./cmd/app
```

This builds the `cmd/app` entry point into `bin/app`.

### 2. Environment setup

No environment variables are needed to run the program. See [Environment variables](#environment-variables) for the variables listed in the migration configuration.

### 3. Database setup

None. The balance is kept in memory, and no schema or migration step exists.

### 4. Run

```sh
./bin/app
```

The program does not listen on any port, even though the migration configuration mentions port 8080. It is an interactive console that shows a menu and waits for a choice on standard input:

- Enter the number of a menu option and press Enter.
- Choice `4` exits with the message `Exiting the program. Goodbye!` and status 0.
- Sending end-of-input (Ctrl+D) at the menu prompt also exits cleanly (see [Migration notes](#migration-notes)).

**First request to try:** none has been verified against the original program. Because this is not an HTTP service, there is no request to send. Start the binary and use the menu.

#### Alternative entry point: `cmd/accountsystem`

A second entry point, `cmd/accountsystem/main.go`, runs the same console with one difference. It handles Ctrl+C (SIGINT) and exits with status `130` when interrupted. `cmd/app` does not install a signal handler.

This entry point was not built or run during migration. It can be started with:

```sh
go run ./cmd/accountsystem
```

## Running tests

```sh
go test ./...
```

The only test file in the project is `internal/money/money_test.go`. The `account`, `cli` and `store` packages have no tests.

## Environment variables

The migration configuration listed the variables below. **None of them are read by the entry points (`cmd/app/main.go`, `cmd/accountsystem/main.go`) or by `internal/store/store.go`.** The program runs without them.

| Variable | Listed purpose | Used by the current code? |
|---|---|---|
| `DATABASE_URL` | Database connection string | No. Storage is in-memory only. |
| `PORT` | HTTP listen port (8080 mentioned) | No. The program does not listen on a port. |
| `JWT_SECRET` | Token signing secret | No. There is no authentication. |

These are leftovers from a generic sandbox configuration: the code never reads them, so they can be omitted from any deployment configuration.

## Architecture overview

```
cmd/
  app/main.go            Entry point: wires store → service → CLI; exits 1 on error
  accountsystem/main.go  Same wiring, plus SIGINT handling (exit 130)
internal/
  money/money.go         Amount type (integer cents), InitialBalance, range validation
  money/money_test.go    Unit tests for the money package
  store/store.go         BalanceStore interface and in-memory MemoryStore (COBOL DataProgram)
  account/service.go     Account operations built on a BalanceStore
  cli/menu.go            Interactive menu loop over stdin/stdout
go.mod                   Module "accountsystem", go 1.22
```

### Startup flow (both entry points)

1. `store.NewMemoryStore(money.InitialBalance)` creates the balance store.
2. `account.NewService(balances)` builds the service on top of it.
3. `cli.Run(ctx, os.Stdin, os.Stdout, svc)` runs the menu loop until the user exits.

### Key details of `internal/store`

- **Interface.** `BalanceStore` has `Read(ctx)` and `Write(ctx, amount)`. `MemoryStore` implements it and is safe for concurrent use.
- **Range check.** `Write` rejects any amount outside `0 .. 999999.99`, the range of COBOL `PIC 9(6)V99`. It returns `ErrOutOfRange` and leaves the stored balance unchanged.
- **COBOL call compatibility.** `store.Call(ctx, s, op, &balance)` reproduces the COBOL `CALL 'DataProgram' USING op, balance` interface:
  - `op` is treated like `PIC X(6)`: it is truncated to 6 characters and trailing spaces are trimmed.
  - `"READ"` copies the stored balance into `*balance`.
  - `"WRITE"` stores `*balance`.
  - Any other code is a no-op that returns `nil`, as in the original.
- **Money representation.** Amounts are integer cents (the `Amount` type in `internal/money/money.go`), not floating point.

## Migration notes

Changes from the original COBOL codebase:

1. **EOF at the menu prompt now exits cleanly.**
   - *Change:* end of input at the menu prompt is treated as choice 4. The program prints `Exiting the program. Goodbye!` and exits with status 0.
   - *Original behaviour:* GnuCOBOL leaves `USER-CHOICE` unchanged on EOF and loops forever.
   - *Reason:* this was a deliberate deviation in the migration plan. The original's EOF behaviour is an infinite loop, so the Go version ends the session cleanly instead.

2. **Explicit range validation on balance writes.**
   - *Change:* `Write` on `MemoryStore` in `internal/store/store.go` returns `ErrOutOfRange` for values below 0 or above 999999.99.
   - *Original behaviour:* the caller's field and the storage field shared the same `PIC 9(6)V99`, so an out-of-range value could never reach DataProgram.
   - *Reason:* Go callers can pass any value, so the value is validated rather than silently truncated. This is marked with a `MIGRATION_NOTE` in `internal/store/store.go`.

### Storage

Storage is unchanged in kind. The original kept the balance in DataProgram's `WORKING-STORAGE`, and the Go version keeps it in process memory. Neither version persists data between runs.

## Known limitations

- **Behavioural equivalence with the COBOL original is unconfirmed.** The comparison run was inconclusive.
- **No persistence.** The balance resets to `money.InitialBalance` on every start, matching the original.
- **Limited test coverage.** Only `internal/money` has unit tests. `internal/account`, `internal/cli` and `internal/store` are untested.
- **Unused configuration.** The listed environment variables (`DATABASE_URL`, `PORT`, `JWT_SECRET`) and the port 8080 reference are not used by the code.
- **Two entry points.** `cmd/app` and `cmd/accountsystem` duplicate the same wiring and differ only in signal handling. Only `cmd/app` was built during migration.

## Manual review required

The migration flagged these COBOL modules as low confidence: `Operations` and `MainProgram`. Verify the Go code that replaces them against the original.

- [ ] **`internal/account/service.go`** (from COBOL `Operations`): check each operation for:
  - correct balance arithmetic in cents;
  - handling of amounts that would make the balance negative or exceed 999999.99;
  - matching any insufficient-funds behaviour in the original.
- [ ] **`internal/cli/menu.go`** (from COBOL `MainProgram`): check that menu text, prompts, output formatting of amounts, and handling of invalid choices match the original. Confirm the EOF-as-choice-4 deviation is acceptable.
- [ ] **`internal/money/money.go`**: confirm `InitialBalance` equals the original COBOL initial value and that display formatting matches COBOL's edited output.
- [ ] **`cmd/app/main.go` vs `cmd/accountsystem/main.go`**: decide which entry point to keep, and remove or document the other.
- [ ] **Behavioural comparison**: run the original COBOL program and `./bin/app` side by side with the same input sequences and compare their output, since the automated comparison was inconclusive.
