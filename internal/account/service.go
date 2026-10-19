// Package account implements the account operations of the original COBOL
// "Operations" subprogram: TOTAL (show balance), CREDIT (add an amount) and
// DEBIT (subtract an amount when funds are sufficient).
//
// The core methods (Balance, Credit, Debit) are I/O-free and operate on
// money.Amount values. Run reproduces the original console dialogue for
// callers that still dispatch on a 6-character operation code.
package account

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"accountsystem/internal/money"
	"accountsystem/internal/store"
)

// Operation codes, matching the COBOL PASSED-OPERATION PIC X(6) values
// (trailing spaces are insignificant).
const (
	OpTotal  = "TOTAL"
	OpCredit = "CREDIT"
	OpDebit  = "DEBIT"
)

// ErrInsufficientFunds is returned by Debit when the balance is lower than
// the requested amount. The stored balance is left unchanged.
var ErrInsufficientFunds = errors.New("account: insufficient funds")

// ErrInvalidAmount is returned when an amount cannot be represented by
// PIC 9(6)V99 (negative or greater than 999999.99).
var ErrInvalidAmount = errors.New("account: amount out of range for PIC 9(6)V99")

// ErrUnknownOperation is returned by Run for an unrecognised operation code.
var ErrUnknownOperation = errors.New("account: unknown operation")

// Service performs account operations against a BalanceStore.
type Service struct {
	store store.BalanceStore

	// lastAmount mirrors the non-INITIAL WORKING-STORAGE field AMOUNT,
	// which keeps its value between calls (relevant when ACCEPT hits EOF).
	lastAmount money.Amount
}

// NewService returns a Service backed by s.
func NewService(s store.BalanceStore) (*Service, error) {
	if s == nil {
		return nil, errors.New("account: nil BalanceStore")
	}
	return &Service{store: s}, nil
}

// Balance returns the current balance (COBOL TOTAL).
func (svc *Service) Balance(ctx context.Context) (money.Amount, error) {
	bal, err := svc.store.Read(ctx)
	if err != nil {
		return 0, fmt.Errorf("account: read balance: %w", err)
	}
	return bal, nil
}

// Credit adds amount to the balance and returns the new balance
// (COBOL CREDIT).
//
// The addition mirrors COBOL "ADD AMOUNT TO FINAL-BALANCE" without
// ON SIZE ERROR: overflowing digits are silently truncated, so
// 999999.99 + 1.00 yields 000000.99.
func (svc *Service) Credit(ctx context.Context, amount money.Amount) (money.Amount, error) {
	if err := validAmount(amount); err != nil {
		return 0, err
	}
	bal, err := svc.Balance(ctx)
	if err != nil {
		return 0, err
	}
	bal = bal.Add(amount)
	if err := svc.store.Write(ctx, bal); err != nil {
		return 0, fmt.Errorf("account: write balance: %w", err)
	}
	return bal, nil
}

// Debit subtracts amount from the balance when the balance is at least
// amount, and returns the new balance (COBOL DEBIT). Otherwise it returns
// ErrInsufficientFunds together with the unchanged balance and does not
// write.
func (svc *Service) Debit(ctx context.Context, amount money.Amount) (money.Amount, error) {
	if err := validAmount(amount); err != nil {
		return 0, err
	}
	bal, err := svc.Balance(ctx)
	if err != nil {
		return 0, err
	}
	if bal < amount {
		return bal, ErrInsufficientFunds
	}
	bal = bal.Sub(amount)
	if err := svc.store.Write(ctx, bal); err != nil {
		return 0, fmt.Errorf("account: write balance: %w", err)
	}
	return bal, nil
}

// Run reproduces the console behaviour of the COBOL Operations program for
// one operation code. Prompts and results are written to out; amounts are
// read (one line each) from in.
//
// op is treated like PIC X(6): only the first six characters count and
// trailing spaces are ignored. An unknown code produces no output and
// returns ErrUnknownOperation (the COBOL program silently did nothing).
func (svc *Service) Run(ctx context.Context, op string, in *bufio.Reader, out io.Writer) error {
	if len(op) > 6 {
		op = op[:6]
	}
	switch strings.TrimRight(op, " ") {
	case OpTotal:
		bal, err := svc.Balance(ctx)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "Current balance: %s\n", bal)
		return err

	case OpCredit:
		if _, err := fmt.Fprintln(out, "Enter credit amount: "); err != nil {
			return err
		}
		amt := svc.accept(in)
		bal, err := svc.Credit(ctx, amt)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "Amount credited. New balance: %s\n", bal)
		return err

	case OpDebit:
		if _, err := fmt.Fprintln(out, "Enter debit amount: "); err != nil {
			return err
		}
		amt := svc.accept(in)
		bal, err := svc.Debit(ctx, amt)
		switch {
		case errors.Is(err, ErrInsufficientFunds):
			_, err = fmt.Fprintln(out, "Insufficient funds for this debit.")
			return err
		case err != nil:
			return err
		}
		_, err = fmt.Fprintf(out, "Amount debited. New balance: %s\n", bal)
		return err
	}
	return fmt.Errorf("%w: %q", ErrUnknownOperation, op)
}

// accept reads one line from in into AMOUNT. On EOF with no data the
// previous AMOUNT value is kept, as with a non-INITIAL COBOL program.
func (svc *Service) accept(in *bufio.Reader) money.Amount {
	if in == nil {
		return svc.lastAmount
	}
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		return svc.lastAmount
	}
	svc.lastAmount = money.ParseAccept(line)
	return svc.lastAmount
}

func validAmount(a money.Amount) error {
	if !a.Valid() {
		return fmt.Errorf("%w: %d cents", ErrInvalidAmount, int64(a))
	}
	return nil
}
