package main

import (
	"context"
	"fmt"
	"os"

	"migrated-app/internal/account"
	"migrated-app/internal/cli"
	"migrated-app/internal/store"
)

// run executes the program with its command-line arguments and returns
// its exit code.
func run(args []string) int {
	_ = args

	ctx := context.Background()

	balances := store.NewMemoryStore()
	svc, err := account.NewService(balances)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := cli.Run(ctx, os.Stdin, os.Stdout, svc); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:]))
}