// Package money models the COBOL "PIC 9(6)V99" unsigned numeric field as
// an integer count of cents.
package money

import (
	"fmt"
	"strings"
)

// Amount is a monetary value in integer cents.
type Amount int64

// Max is the largest value PIC 9(6)V99 can hold (999999.99).
const Max Amount = 99999999

// InitialBalance is the starting balance (1000.00), from
// "STORAGE-BALANCE PIC 9(6)V99 VALUE 1000.00".
const InitialBalance Amount = 100000

// modulus is 10^8: PIC 9(6)V99 holds 8 decimal digits of cents.
const modulus = Max + 1

// normalize maps a into the representable range 0..Max, discarding
// high-order digits and the sign as an unsigned PIC field would.
func normalize(a Amount) Amount {
	return ((a % modulus) + modulus) % modulus
}

// String renders the amount zero-padded with a visible decimal point,
// e.g. 1000.00 -> "001000.00".
func (a Amount) String() string {
	n := normalize(a)
	return fmt.Sprintf("%06d.%02d", int64(n/100), int64(n%100))
}

// Valid reports whether a fits PIC 9(6)V99 (0..999999.99).
func (a Amount) Valid() bool {
	return a >= 0 && a <= Max
}

// Add adds b to a, silently truncating high-order digits that do not fit
// (COBOL ADD without ON SIZE ERROR): 999999.99 + 1.00 = 000000.99.
func (a Amount) Add(b Amount) Amount {
	return normalize(a + b)
}

// Sub subtracts b from a. Callers are expected to check a >= b first, as
// the COBOL program does; a negative result loses its sign like an
// unsigned PIC field.
func (a Amount) Sub(b Amount) Amount {
	return normalize(a - b)
}

// ParseAccept converts console input to an Amount following ACCEPT into an
// unsigned PIC 9(6)V99 field. It never fails: surrounding blanks and any
// sign are ignored, the integer part keeps its low-order six digits, the
// fraction keeps two digits (truncated), and non-numeric input yields 0.
//
// MIGRATION_NOTE: GnuCOBOL's exact ACCEPT conversion for malformed input is
// runtime-dependent; this approximation must be checked against the golden
// transcripts captured from the original program.
func ParseAccept(s string) Amount {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "+-")
	s = strings.TrimRight(s, "+-")
	intPart, fracPart, _ := strings.Cut(s, ".")
	if intPart == "" && fracPart == "" {
		return 0
	}
	if !allDigits(intPart) || !allDigits(fracPart) {
		return 0
	}
	if len(intPart) > 6 {
		intPart = intPart[len(intPart)-6:]
	}
	var whole Amount
	for _, r := range intPart {
		whole = whole*10 + Amount(r-'0')
	}
	fracPart += "00"
	frac := Amount(fracPart[0]-'0')*10 + Amount(fracPart[1]-'0')
	return whole*100 + frac
}

// ParseChoice converts one input line into the value ACCEPT would store in
// a "PIC 9" field. Surrounding blanks are ignored and the low-order digit
// of numeric input is kept, so "12" becomes 2. Empty or non-numeric input
// yields 0.
func ParseChoice(s string) int {
	s = strings.TrimSpace(s)
	if s == "" || !allDigits(s) {
		return 0
	}
	return int(s[len(s)-1] - '0')
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
