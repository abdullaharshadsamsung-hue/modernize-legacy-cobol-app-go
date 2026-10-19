// Command accountsystem runs the interactive account management console
// migrated from the COBOL MainProgram/Operations/DataProgram programs.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"accountsystem/internal/account"
	"accountsystem/internal/cli"
	"accountsystem/internal/money"
	"accountsystem/internal/store"
)

// run executes the program with its command-line arguments and returns
// its exit code.
func run(args []string) int {
	_ = args

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	balances := store.NewMemoryStore(money.InitialBalance)
	svc, err := account.NewService(balances)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := cli.Run(ctx, os.Stdin, os.Stdout, svc); err != nil {
		if errors.Is(err, context.Canceled) {
			return 130
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:]))
}
