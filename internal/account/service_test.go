package account

import (
	"bufio"
	"context"
	"errors"
	"strings"
	"testing"

	"migrated-app/internal/store"
)

type fakeStore struct {
	balance  int64
	readErr  error
	writeErr error
	calls    []string
	writes   []int64
}

func (f *fakeStore) Read(ctx context.Context) (int64, error) {
	f.calls = append(f.calls, "READ")
	if f.readErr != nil {
		return 0, f.readErr
	}
	return f.balance, nil
}

func (f *fakeStore) Write(ctx context.Context, v int64) error {
	f.calls = append(f.calls, "WRITE")
	if f.writeErr != nil {
		return f.writeErr
	}
	f.balance = v
	f.writes = append(f.writes, v)
	return nil
}

var _ store.BalanceStore = (*fakeStore)(nil)

func newSvc(t *testing.T, fs *fakeStore) *Service {
	t.Helper()
	svc, err := NewService(fs)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func eqCalls(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestNewServiceNil(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name      string
		op        string
		balance   int64
		input     string
		wantOut   string
		wantBal   int64
		wantCalls []string
		wantErr   error
		wantRest  string
	}{
		{"total", "TOTAL ", 100000, "untouched\n",
			"Current balance: 00100000\n", 100000, []string{"READ"}, nil, "untouched\n"},
		{"credit valid", "CREDIT", 100000, "250.00\n",
			"Enter credit amount: \nAmount credited. New balance: 00125000\n", 125000, []string{"READ", "WRITE"}, nil, ""},
		{"credit overflow truncates", "CREDIT", 99999999, "1.00\n",
			"Enter credit amount: \nAmount credited. New balance: 00000099\n", 99, []string{"READ", "WRITE"}, nil, ""},
		{"debit equal", "DEBIT ", 100000, "1000.00\n",
			"Enter debit amount: \nAmount debited. New balance: 00000000\n", 0, []string{"READ", "WRITE"}, nil, ""},
		{"debit partial", "DEBIT ", 100000, "0.50\n",
			"Enter debit amount: \nAmount debited. New balance: 00099950\n", 99950, []string{"READ", "WRITE"}, nil, ""},
		{"debit insufficient", "DEBIT ", 10000, "100.01\n",
			"Enter debit amount: \nInsufficient funds for this debit.\n", 10000, []string{"READ"}, nil, ""},
		{"lowercase unknown", "total ", 100000, "x\n", "", 100000, nil, ErrUnknownOperation, "x\n"},
		{"exit unknown", "EXIT  ", 100000, "x\n", "", 100000, nil, ErrUnknownOperation, "x\n"},
		{"blanks unknown", "      ", 100000, "x\n", "", 100000, nil, ErrUnknownOperation, "x\n"},
		{"empty unknown", "", 100000, "x\n", "", 100000, nil, ErrUnknownOperation, "x\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := &fakeStore{balance: tc.balance}
			svc := newSvc(t, fs)
			in := bufio.NewReader(strings.NewReader(tc.input))
			var out strings.Builder
			op := tc.op
			err := svc.Run(context.Background(), op, in, &out)
			if tc.wantErr == nil && err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if out.String() != tc.wantOut {
				t.Errorf("out = %q, want %q", out.String(), tc.wantOut)
			}
			if fs.balance != tc.wantBal {
				t.Errorf("balance = %d, want %d", fs.balance, tc.wantBal)
			}
			if !eqCalls(fs.calls, tc.wantCalls) {
				t.Errorf("calls = %v, want %v", fs.calls, tc.wantCalls)
			}
			rest, _ := in.ReadString(0)
			if rest != tc.wantRest {
				t.Errorf("remaining input = %q, want %q", rest, tc.wantRest)
			}
			if op != tc.op {
				t.Errorf("op modified")
			}
			if fs.balance < 0 {
				t.Errorf("negative balance")
			}
		})
	}
}

func TestRunStoreErrors(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name  string
		op    string
		fs    *fakeStore
		input string
	}{
		{"total read err", "TOTAL ", &fakeStore{readErr: boom}, ""},
		{"credit read err", "CREDIT", &fakeStore{readErr: boom}, "1.00\n"},
		{"credit write err", "CREDIT", &fakeStore{writeErr: boom}, "1.00\n"},
		{"debit read err", "DEBIT ", &fakeStore{readErr: boom}, "1.00\n"},
		{"debit write err", "DEBIT ", &fakeStore{balance: 500, writeErr: boom}, "1.00\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newSvc(t, tc.fs)
			var out strings.Builder
			err := svc.Run(context.Background(), tc.op, bufio.NewReader(strings.NewReader(tc.input)), &out)
			if !errors.Is(err, boom) {
				t.Fatalf("err = %v, want boom", err)
			}
		})
	}
}

func TestCreditDebitDirect(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		debit   bool
		balance int64
		amount  int64
		want    int64
		wantErr error
		writes  int
	}{
		{"credit", false, 100000, 25000, 125000, nil, 1},
		{"credit max", false, 0, store.MaxBalanceCents, store.MaxBalanceCents, nil, 1},
		{"credit negative", false, 100, -1, 0, ErrInvalidAmount, 0},
		{"credit too large", false, 100, store.MaxBalanceCents + 1, 0, ErrInvalidAmount, 0},
		{"debit", true, 100000, 100000, 0, nil, 1},
		{"debit insufficient", true, 10000, 10001, 10000, ErrInsufficientFunds, 0},
		{"debit negative", true, 100, -5, 0, ErrInvalidAmount, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := &fakeStore{balance: tc.balance}
			svc := newSvc(t, fs)
			var got int64
			var err error
			if tc.debit {
				got, err = svc.Debit(ctx, tc.amount)
			} else {
				got, err = svc.Credit(ctx, tc.amount)
			}
			if tc.wantErr == nil && err != nil {
				t.Fatalf("unexpected err %v", err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
			if len(fs.writes) != tc.writes {
				t.Errorf("writes = %d, want %d", len(fs.writes), tc.writes)
			}
			if tc.writes > 0 && (len(fs.calls) < 2 || fs.calls[0] != "READ") {
				t.Errorf("write not preceded by read: %v", fs.calls)
			}
		})
	}
}

func TestBalance(t *testing.T) {
	fs := &fakeStore{balance: 4242}
	svc := newSvc(t, fs)
	got, err := svc.Balance(context.Background())
	if err != nil || got != 4242 {
		t.Fatalf("got %d, %v", got, err)
	}
}

func TestFormatPIC(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "00000000"},
		{100000, "00100000"},
		{125000, "00125000"},
		{99999999, "99999999"},
		{100000000, "00000000"},
		{100000099, "00000099"},
		{-1, "99999999"},
	}
	for _, tc := range tests {
		if got := FormatPIC(tc.in); got != tc.want {
			t.Errorf("FormatPIC(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"250.00", 25000},
		{"250", 25000},
		{" 1000.00 \n", 100000},
		{"0.5", 50},
		{"1.999", 199},
		{".75", 75},
		{"5.", 500},
		{"1234567.89", 23456789},
		{"-100.00", 10000},
		{"100.00-", 10000},
		{"+3", 300},
		{"abc", 0},
		{"12a", 0},
		{"", 0},
		{".", 0},
		{"999999.99", 99999999},
	}
	for _, tc := range tests {
		if got := ParseAmount(tc.in); got != tc.want {
			t.Errorf("ParseAmount(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestAcceptEOFKeepsLastAmount(t *testing.T) {
	fs := &fakeStore{balance: 0}
	svc := newSvc(t, fs)
	ctx := context.Background()
	var out strings.Builder
	if err := svc.Run(ctx, "CREDIT", bufio.NewReader(strings.NewReader("5.00\n")), &out); err != nil {
		t.Fatal(err)
	}
	if err := svc.Run(ctx, "CREDIT", bufio.NewReader(strings.NewReader("")), &out); err != nil {
		t.Fatal(err)
	}
	if fs.balance != 1000 {
		t.Errorf("balance = %d, want 1000", fs.balance)
	}
	if err := svc.Run(ctx, "CREDIT", nil, &out); err != nil {
		t.Fatal(err)
	}
	if fs.balance != 1500 {
		t.Errorf("balance = %d, want 1500", fs.balance)
	}
	// line without trailing newline is still accepted
	if err := svc.Run(ctx, "DEBIT", bufio.NewReader(strings.NewReader("15.00")), &out); err != nil {
		t.Fatal(err)
	}
	if fs.balance != 0 {
		t.Errorf("balance = %d, want 0", fs.balance)
	}
}

func TestRunOpTruncatedToSixChars(t *testing.T) {
	fs := &fakeStore{balance: 100}
	svc := newSvc(t, fs)
	var out strings.Builder
	if err := svc.Run(context.Background(), "TOTAL XYZ", nil, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Current balance: 00000100\n" {
		t.Errorf("out = %q", out.String())
	}
}