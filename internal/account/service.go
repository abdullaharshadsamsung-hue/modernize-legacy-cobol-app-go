// Package account implements the account operations of the original COBOL
// "Operations" subprogram: TOTAL (show balance), CREDIT (add an amount) and
// DEBIT (subtract an amount when funds are sufficient).
//
// The core methods (Balance, Credit, Debit) are I/O-free and operate on
// integer cents. Run reproduces the original console dialogue for callers
// that still dispatch on a 6-character operation code.
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

// maxBalanceCents is the largest value PIC 9(6)V99 can hold (999999.99).
const maxBalanceCents = int64(money.Max)

// Operation codes, matching the COBOL PASSED-OPERATION PIC X(6) values
// (trailing spaces are insignificant).
const (
	OpTotal  = "TOTAL"
	OpCredit = "CREDIT"
	OpDebit  = "DEBIT"
)

// picModulus is 10^8: PIC 9(6)V99 holds 8 decimal digits of cents.
const picModulus = maxBalanceCents + 1

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
	lastAmount int64
}

// NewService returns a Service backed by s.
func NewService(s store.BalanceStore) (*Service, error) {
	if s == nil {
		return nil, errors.New("account: nil BalanceStore")
	}
	return &Service{store: s}, nil
}

// Balance returns the current balance in cents (COBOL TOTAL).
func (svc *Service) Balance(ctx context.Context) (int64, error) {
	bal, err := svc.store.Read(ctx)
	if err != nil {
		return 0, fmt.Errorf("account: read balance: %w", err)
	}
	return bal, nil
}

// Credit adds amount cents to the balance and returns the new balance
// (COBOL CREDIT).
//
// The addition mirrors COBOL "ADD AMOUNT TO FINAL-BALANCE" without
// ON SIZE ERROR: overflowing digits are silently truncated, so
// 999999.99 + 1.00 yields 000000.99.
func (svc *Service) Credit(ctx context.Context, amount int64) (int64, error) {
	if err := validAmount(amount); err != nil {
		return 0, err
	}
	bal, err := svc.Balance(ctx)
	if err != nil {
		return 0, err
	}
	bal = addTruncating(bal, amount)
	if err := svc.store.Write(ctx, bal); err != nil {
		return 0, fmt.Errorf("account: write balance: %w", err)
	}
	return bal, nil
}

// Debit subtracts amount cents from the balance when the balance is at
// least amount, and returns the new balance (COBOL DEBIT). Otherwise it
// returns ErrInsufficientFunds together with the unchanged balance and
// does not write.
func (svc *Service) Debit(ctx context.Context, amount int64) (int64, error) {
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
	bal -= amount
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
		_, err = fmt.Fprintf(out, "Current balance: %s\n", FormatPIC(bal))
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
		_, err = fmt.Fprintf(out, "Amount credited. New balance: %s\n", FormatPIC(bal))
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
		_, err = fmt.Fprintf(out, "Amount debited. New balance: %s\n", FormatPIC(bal))
		return err
	}
	return fmt.Errorf("%w: %q", ErrUnknownOperation, op)
}

// FormatPIC renders cents the way COBOL DISPLAYs an unsigned PIC 9(6)V99
// field: eight zero-padded digits with no decimal point (1000.00 -> "00100000").
func FormatPIC(cents int64) string {
	return fmt.Sprintf("%08d", ((cents%picModulus)+picModulus)%picModulus)
}

// ParseAmount converts console input to cents following ACCEPT into an
// unsigned PIC 9(6)V99 field. It never fails: surrounding blanks and any
// sign are ignored, the integer part keeps its low-order six digits, the
// fraction keeps two digits (truncated), and non-numeric input yields 0.
//
// MIGRATION_NOTE: GnuCOBOL's exact ACCEPT conversion for malformed input is
// runtime-dependent; this approximation must be checked against the golden
// transcripts captured from the original program.
func ParseAmount(s string) int64 {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "+-")
	s = strings.TrimRight(s, "+-")
	intPart, fracPart, _ := strings.Cut(s, ".")
	if intPart == "" && fracPart == "" {
		return 0
	}
	for _, part := range []string{intPart, fracPart} {
		for _, r := range part {
			if r < '0' || r > '9' {
				return 0
			}
		}
	}
	var whole int64
	if len(intPart) > 6 {
		intPart = intPart[len(intPart)-6:]
	}
	for _, r := range intPart {
		whole = whole*10 + int64(r-'0')
	}
	fracPart += "00"
	frac := int64(fracPart[0]-'0')*10 + int64(fracPart[1]-'0')
	return whole*100 + frac
}

// accept reads one line from in into AMOUNT. On EOF with no data the
// previous AMOUNT value is kept, as with a non-INITIAL COBOL program.
func (svc *Service) accept(in *bufio.Reader) int64 {
	if in == nil {
		return svc.lastAmount
	}
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		return svc.lastAmount
	}
	svc.lastAmount = ParseAmount(line)
	return svc.lastAmount
}

// addTruncating adds two PIC 9(6)V99 values, discarding high-order digits
// that do not fit (COBOL ADD without ON SIZE ERROR).
func addTruncating(a, b int64) int64 {
	return (a + b) % picModulus
}

func validAmount(cents int64) error {
	if cents < 0 || cents > store.MaxBalanceCents {
		return fmt.Errorf("%w: %d cents", ErrInvalidAmount, cents)
	}
	return nil
}
