// Package cli implements the interactive console menu of the original COBOL
// MainProgram: it repeatedly shows a four-option menu, reads a one-digit
// choice and dispatches to the account operations until the user exits.
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"

	"accountsystem/internal/account"
	"accountsystem/internal/money"
)

// Operator runs one account operation identified by a 6-character COBOL
// operation code ("TOTAL ", "CREDIT", "DEBIT "). *account.Service
// satisfies it.
type Operator interface {
	Run(ctx context.Context, op string, in *bufio.Reader, out io.Writer) error
}

// Compile-time check that the migrated account service fits the menu.
var _ Operator = (*account.Service)(nil)

// Menu choices, matching the EVALUATE USER-CHOICE branches.
const (
	ChoiceViewBalance = 1
	ChoiceCredit      = 2
	ChoiceDebit       = 3
	ChoiceExit        = 4
)

// Operation codes passed BY REFERENCE to CALL 'Operations' (PIC X(6)
// literals, trailing spaces preserved).
const (
	opTotal  = "TOTAL "
	opCredit = "CREDIT"
	opDebit  = "DEBIT "
)

const separator = "--------------------------------"

// menuLines is the fixed menu DISPLAYed on every loop iteration.
var menuLines = []string{
	separator,
	"Account Management System",
	"1. View Balance",
	"2. Credit Account",
	"3. Debit Account",
	"4. Exit",
	separator,
	"Enter your choice (1-4): ",
}

// ParseChoice converts one input line into the value ACCEPT would store in
// "USER-CHOICE PIC 9". Surrounding blanks are ignored and, as GnuCOBOL
// truncates ACCEPT input to the size of a numeric DISPLAY field, only the
// first character is kept (so "12" becomes 1); a non-digit yields 0, an
// invalid choice.
func ParseChoice(line string) int {
	s := strings.TrimSpace(line)
	if s == "" || s[0] < '0' || s[0] > '9' {
		return 0
	}
	return int(s[0] - '0')
}

// Run drives the menu loop, reading choices from in and writing all output
// to out, until the user selects option 4. It returns nil on a normal exit,
// ctx.Err() if the context is cancelled, or the first I/O or operation
// error encountered.
//
// Deliberate deviation: end of input is treated as choice 4 instead of
// looping forever as the COBOL program would.
func Run(ctx context.Context, in io.Reader, out io.Writer, ops Operator) error {
	if ops == nil {
		return errors.New("cli: nil Operator")
	}
	reader := bufio.NewReader(in)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := printMenu(out); err != nil {
			return fmt.Errorf("cli: write menu: %w", err)
		}

		choice, err := readChoice(reader)
		if err != nil {
			return fmt.Errorf("cli: read choice: %w", err)
		}

		switch choice {
		case ChoiceViewBalance:
			err = ops.Run(ctx, opTotal, reader, out)
		case ChoiceCredit:
			err = ops.Run(ctx, opCredit, reader, out)
		case ChoiceDebit:
			err = ops.Run(ctx, opDebit, reader, out)
		case ChoiceExit:
			if _, err := fmt.Fprintln(out, "Exiting the program. Goodbye!"); err != nil {
				return fmt.Errorf("cli: write goodbye: %w", err)
			}
			return nil
		default:
			_, err = fmt.Fprintln(out, "Invalid choice, please select 1-4.")
		}
		if err != nil {
			return fmt.Errorf("cli: choice %d: %w", choice, err)
		}
	}
}

func printMenu(out io.Writer) error {
	for _, line := range menuLines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

// readChoice reads one line and parses it. A clean EOF with no pending data
// maps to ChoiceExit (see Run).
func readChoice(r *bufio.Reader) (int, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			if line == "" {
				return ChoiceExit, nil
			}
			return ParseChoice(line), nil
		}
		return 0, err
	}
	return ParseChoice(line), nil
}
