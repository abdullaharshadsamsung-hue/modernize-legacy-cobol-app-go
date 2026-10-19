// Package store holds the account balance that the original COBOL
// DataProgram subprogram kept in WORKING-STORAGE.
//
// The balance lives in process memory only: it starts at the configured
// initial balance (money.InitialBalance, 1000.00) and resets whenever the
// process restarts, exactly like the COBOL run unit. No database is
// involved, so no schema is created.
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"accountsystem/internal/money"
)

// ErrOutOfRange is returned when a balance to be written cannot be
// represented by PIC 9(6)V99 (negative or greater than 999999.99).
var ErrOutOfRange = errors.New("store: balance out of range for PIC 9(6)V99")

// BalanceStore reads and writes the single stored account balance.
type BalanceStore interface {
	// Read returns the stored balance.
	Read(ctx context.Context) (money.Amount, error)
	// Write replaces the stored balance.
	Write(ctx context.Context, amount money.Amount) error
}

// MemoryStore is a concurrency-safe, in-memory BalanceStore.
type MemoryStore struct {
	mu      sync.Mutex
	balance money.Amount
}

// Compile-time check that MemoryStore satisfies BalanceStore.
var _ BalanceStore = (*MemoryStore)(nil)

// NewMemoryStore returns a MemoryStore initialised to initial.
func NewMemoryStore(initial money.Amount) *MemoryStore {
	return &MemoryStore{balance: initial}
}

// Read returns the stored balance (COBOL 'READ': MOVE STORAGE-BALANCE TO BALANCE).
func (s *MemoryStore) Read(ctx context.Context) (money.Amount, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.balance, nil
}

// Write stores amount as the new balance (COBOL 'WRITE': MOVE BALANCE TO
// STORAGE-BALANCE). Values outside 0..999999.99 are rejected with
// ErrOutOfRange and leave the stored balance unchanged.
//
// MIGRATION_NOTE: in COBOL the caller's field shared the same PIC, so an
// out-of-range value could never reach this point. Go callers can pass any
// value, so it is validated explicitly rather than silently truncated.
func (s *MemoryStore) Write(ctx context.Context, amount money.Amount) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !amount.Valid() {
		return fmt.Errorf("%w: %d cents", ErrOutOfRange, int64(amount))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.balance = amount
	return nil
}

// Operation codes accepted by Call, matching the COBOL PASSED-OPERATION values.
const (
	OpRead  = "READ"
	OpWrite = "WRITE"
)

// Call reproduces the COBOL "CALL 'DataProgram' USING op, balance" interface
// for callers that still dispatch on an operation code.
//
// op is treated like PIC X(6): only the first six characters count and
// trailing spaces are ignored. "READ" copies the stored balance into
// *balance; "WRITE" stores *balance. Any other code does nothing and
// returns nil, as in the original.
func Call(ctx context.Context, s BalanceStore, op string, balance *money.Amount) error {
	if s == nil {
		return errors.New("store: nil BalanceStore")
	}
	if balance == nil {
		return errors.New("store: nil balance pointer")
	}
	if len(op) > 6 {
		op = op[:6]
	}
	switch strings.TrimRight(op, " ") {
	case OpRead:
		v, err := s.Read(ctx)
		if err != nil {
			return fmt.Errorf("store: read: %w", err)
		}
		*balance = v
	case OpWrite:
		if err := s.Write(ctx, *balance); err != nil {
			return fmt.Errorf("store: write: %w", err)
		}
	}
	return nil
}
