# Copilot COBOL Account App (Go)

An interactive, menu-driven console program for a single account balance. You can view the balance, credit it and debit it. It is a Go migration of a COBOL application made of three programs: `MainProgram`, `Operations` and `DataProgram`.

The program reads menu choices and amounts from standard input and writes results to standard output. The balance is held in process memory only. It starts at `money.InitialBalance` and resets every time the program starts, as the original COBOL run unit did. There is no database and no persistence.

## Migration status

| Check | Result |
|---|---|
| Modules migrated | 3/3 |
| Build | Passed |
| Unit tests (`go test ./...`) | Passed |
| Behavior compared with the original COBOL program | Passed, for the sessions listed under [Try it](#try-it) |
| Model self-assessed confidence | 96%. This is the migration tool's own estimate, not a measured result. |

The `Operations` and `MainProgram` modules are flagged as low confidence. See [Manual review required](#manual-review-required).

## Tech stack

- **Language:** Go. The module `accountsystem` declares `go 1.23.0` in `go.mod`.
- **Libraries:** standard library only, plus `github.com/spf13/viper v1.21.0`, which is listed as a direct dependency in `go.mod`. The two entry points do not use viper directly.
- **Storage:** in-memory, in `internal/store`. No database.

## Prerequisites

- Go toolchain. The migration was built and tested with the toolchain from the `golang:1.27-alpine` image (Go 1.27). `go.mod` requires at least Go 1.23.0.
- Network access to the Go module proxy for the first `go mod download`.

No database or other services are required.

## Getting started

Run all commands from the root of a fresh clone.

### 1. Download dependencies and build

```sh
mkdir -p bin
go mod download
go build -o bin/app ./cmd/app
```

This produces the `bin/app` binary from `cmd/app`. That is the entry point used for verification.

### 2. Environment setup

None is required. See [Environment variables](#environment-variables) for why the variables listed by the migration run are not needed.

### 3. Database setup

None. The balance is kept in memory, and the original COBOL program had no database either.

### 4. Run

```sh
./bin/app
```

The program is an interactive console. It does **not** open a network port.

The migration run's metadata stated "listens on port 8080", but it recorded no run command. Neither `cmd/app/main.go` nor `cmd/accountsystem/main.go` starts a server. Both read from stdin and write to stdout.

Choose `4` at the menu, or send end-of-input (Ctrl-D), to exit.

### Try it

The migration verified the following sessions against the original program. Each one runs with no arguments and exits with status `0`. The verified inputs, exactly as recorded, are:

```
{"args": [], "stdin": "1\n4\n"}
{"args": [], "stdin": "2\n500.00\n1\n4\n"}
{"args": [], "stdin": "3\n200.00\n1\n4\n"}
{"args": [], "stdin": "3\n1000.00\n1\n4\n"}
{"args": [], "stdin": "3\n1500.00\n1\n4\n"}
{"args": [], "stdin": "9\n0\n4\n"}
```

To reproduce them from the shell, pipe the stdin string into the binary and check the exit code:

```sh
printf '1\n4\n' | ./bin/app; echo "exit: $?"
printf '2\n500.00\n1\n4\n' | ./bin/app; echo "exit: $?"
printf '3\n200.00\n1\n4\n' | ./bin/app; echo "exit: $?"
printf '3\n1000.00\n1\n4\n' | ./bin/app; echo "exit: $?"
printf '3\n1500.00\n1\n4\n' | ./bin/app; echo "exit: $?"
printf '9\n0\n4\n' | ./bin/app; echo "exit: $?"
```

In these sessions:

- `1` shows the balance.
- `2` and `3` each prompt for an amount (credit and debit), which is entered on the next line.
- `4` exits.
- `9` exercises the invalid-choice path.

### Alternative entry point

`cmd/accountsystem` runs the same console. It also handles Ctrl-C (SIGINT): it cancels the session and exits with status `130`. Build and run it with:

```sh
go build -o bin/accountsystem ./cmd/accountsystem
./bin/accountsystem
```

## Running tests

```sh
go test ./...
```

Tests live next to the code they cover:

- `internal/account/service_test.go`
- `internal/cli/menu_test.go`
- `internal/money/money_test.go`
- `internal/store/store_test.go`

## Environment variables

The migration run's sandbox listed the following variables. None of them is read by either entry point (`cmd/app/main.go`, `cmd/accountsystem/main.go`) or by `internal/store/store.go`.

| Variable | Required | Purpose |
|---|---|---|
| `DATABASE_URL` | No | Not used. The app has no database. The store is in-memory. |
| `PORT` | No | Not used. The app is a stdin/stdout console and does not listen on a port. |
| `JWT_SECRET` | No | Not used. There is no authentication. |

## Architecture overview

```
cmd/
  app/main.go            Entry point used for verification: wires store -> service -> CLI
  accountsystem/main.go  Same wiring plus Ctrl-C handling (exit 130)
internal/
  money/                 Money amount type and InitialBalance
  store/                 Balance storage (replaces COBOL DataProgram)
  account/               Account operations service (replaces COBOL Operations)
  cli/                   Interactive menu loop (replaces COBOL MainProgram)
```

### Startup wiring

Both entry points create the components in the same order:

1. `store.NewMemoryStore(money.InitialBalance)` creates the balance store.
2. `account.NewService(balances)` builds the account service on top of the store.
3. `cli.Run(ctx, os.Stdin, os.Stdout, svc)` runs the menu loop until the user exits.

### Package responsibilities

- **`internal/money`**
  - Defines the amount type in `internal/money/money.go`, an integer count of cents that models COBOL `PIC 9(6)V99`.
  - Defines `InitialBalance`.
  - Provides `Amount.Valid()`, which checks the range `0..999999.99`.
- **`internal/store`**
  - Defines the `BalanceStore` interface, with `Read` and `Write`.
  - `MemoryStore` is a mutex-protected, in-memory implementation.
  - `Write` rejects out-of-range values with `ErrOutOfRange` and leaves the stored balance unchanged.
  - `internal/store/store.go` also reproduces the COBOL `CALL 'DataProgram' USING op, balance` interface:
    - The operation code is treated like `PIC X(6)`: only the first six characters count and trailing spaces are trimmed.
    - `READ` and `WRITE` are supported.
    - Unknown codes are a no-op.
- **`internal/account`**
  - Implements the account operations (view, credit, debit) against a `BalanceStore`.
- **`internal/cli`**
  - Implements the menu prompt, input parsing and output.
  - Takes an `io` reader and writer and a context, which makes it testable without a terminal.

## Migration notes

The original was COBOL with no framework (`MainProgram`, `Operations`, `DataProgram`). The target is standard-library Go.

| Change | Reason |
|---|---|
| End of input (EOF) at the menu prompt is treated as choice `4`. The program prints `Exiting the program. Goodbye!` and exits with status `0`. The original GnuCOBOL program leaves `USER-CHOICE` unchanged on EOF and loops forever. | Deliberate deviation in the migration plan. The original's EOF behavior is an infinite loop, so the Go version ends the session cleanly instead. |
| The in-memory store's `Write` (in `internal/store/store.go`) validates the amount and returns `ErrOutOfRange` for values outside `0..999999.99`. It does not silently truncate. | In COBOL the caller's field shared the same `PIC`, so an out-of-range value could never reach `DataProgram`. Go callers can pass any value, so the range is checked explicitly. This is recorded as a `MIGRATION_NOTE` in `internal/store/store.go`. |
| `DataProgram`'s `WORKING-STORAGE` balance became an in-memory `MemoryStore` behind a `BalanceStore` interface. | Keeps the original semantics: process-local storage that resets on restart. The interface lets the service be tested in isolation. |
| `cmd/accountsystem` exits with status `130` on Ctrl-C. | Go-side addition for clean interrupt handling. The original had no equivalent. |

**Database:** unchanged. Neither the original nor the migrated version uses a database.

## Known limitations

- **Behavior coverage is limited.** Equivalence with the original was compared only for the six sessions listed under [Try it](#try-it). Other inputs are covered only by the unit tests and have not been compared against the COBOL program. Examples include malformed amounts, amounts with more than two decimals, and balances near `999999.99`.
- **No persistence.** The balance resets to `money.InitialBalance` on every start, matching the original.
- **Single user.** One process holds one balance.
- **Misleading run metadata.** The migration run's environment listed `DATABASE_URL`, `PORT` and `JWT_SECRET`, and claimed the app listens on port 8080. The code does not support any of this. Do not deploy it as a network service.
- **Two entry points.** `cmd/app` and `cmd/accountsystem` duplicate the same wiring. Only `cmd/app` was used for build and behavior verification.

## Manual review required

| Original module | Go code to review | What to check |
|---|---|---|
| `Operations` (low confidence) | `internal/account/service.go` | Credit and debit arithmetic, rounding, and the insufficient-funds path. The verified debits of `1000.00` and `1500.00` exercise boundary cases. Also check output wording against the COBOL `DISPLAY` statements. |
| `MainProgram` (low confidence) | `internal/cli/menu.go` | Menu text, prompts, invalid-choice handling (`9` was verified), amount parsing, and the EOF-as-exit deviation. |

Also check the following:

- **`go.mod`:** confirm whether `github.com/spf13/viper` is actually needed. Neither entry point imports it. If no package uses it, remove it with `go mod tidy`.
- **`cmd/app/main.go` vs `cmd/accountsystem/main.go`:** decide which entry point to keep.