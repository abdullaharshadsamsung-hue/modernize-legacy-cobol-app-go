package cli

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type fakeOps struct {
	calls  []string
	output string
	err    error
	failOn string
	read   bool
}

func (f *fakeOps) Run(ctx context.Context, op string, in *bufio.Reader, out io.Writer) error {
	f.calls = append(f.calls, op)
	if f.read {
		if _, err := in.ReadString('\n'); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
	}
	if f.output != "" {
		io.WriteString(out, f.output)
	}
	if f.err != nil && (f.failOn == "" || f.failOn == op) {
		return f.err
	}
	return nil
}

const menuText = "--------------------------------\n" +
	"Account Management System\n" +
	"1. View Balance\n" +
	"2. Credit Account\n" +
	"3. Debit Account\n" +
	"4. Exit\n" +
	"--------------------------------\n" +
	"Enter your choice (1-4): \n"

const goodbye = "Exiting the program. Goodbye!\n"
const invalid = "Invalid choice, please select 1-4.\n"

func TestParseChoice(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"1\n", 1}, {"2", 2}, {"3\r\n", 3}, {"4", 4},
		{"0", 0}, {"5", 5}, {"9", 9},
		{"  2  \n", 2}, {"", 0}, {"   \n", 0},
		{"a", 0}, {"x1", 0}, {"12", 1}, {"-1", 0},
	}
	for _, tt := range tests {
		if got := ParseChoice(tt.in); got != tt.want {
			t.Errorf("ParseChoice(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		opsOut    string
		wantCalls []string
		wantOut   string
	}{
		{"exit", "4\n", "", nil, menuText + goodbye},
		{"view balance", "1\n4\n", "", []string{"TOTAL "}, menuText + menuText + goodbye},
		{"credit", "2\n4\n", "", []string{"CREDIT"}, menuText + menuText + goodbye},
		{"debit", "3\n4\n", "", []string{"DEBIT "}, menuText + menuText + goodbye},
		{"ops output in place", "1\n4\n", "BAL\n", []string{"TOTAL "}, menuText + "BAL\n" + menuText + goodbye},
		{"invalid 0", "0\n4\n", "", nil, menuText + invalid + menuText + goodbye},
		{"invalid 5", "5\n4\n", "", nil, menuText + invalid + menuText + goodbye},
		{"invalid 9", "9\n4\n", "", nil, menuText + invalid + menuText + goodbye},
		{"non numeric", "abc\n4\n", "", nil, menuText + invalid + menuText + goodbye},
		{"empty line", "\n4\n", "", nil, menuText + invalid + menuText + goodbye},
		{"multi char uses first digit", "14\n4\n", "", []string{"TOTAL "}, menuText + menuText + goodbye},
		{"sequence 1 2 3 4", "1\n2\n3\n4\n", "", []string{"TOTAL ", "CREDIT", "DEBIT "},
			strings.Repeat(menuText, 4) + goodbye},
		{"eof treated as exit", "", "", nil, menuText + goodbye},
		{"eof after invalid", "7\n", "", nil, menuText + invalid + menuText + goodbye},
		{"partial line at eof", "2", "", []string{"CREDIT"}, menuText + menuText + goodbye},
		{"exit stops reading", "4\n1\n", "", nil, menuText + goodbye},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops := &fakeOps{output: tt.opsOut}
			var out strings.Builder
			if err := Run(context.Background(), strings.NewReader(tt.input), &out, ops); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.String() != tt.wantOut {
				t.Errorf("output mismatch:\ngot:\n%q\nwant:\n%q", out.String(), tt.wantOut)
			}
			if len(ops.calls) != len(tt.wantCalls) {
				t.Fatalf("calls = %q, want %q", ops.calls, tt.wantCalls)
			}
			for i, c := range ops.calls {
				if c != tt.wantCalls[i] {
					t.Errorf("call %d = %q, want %q", i, c, tt.wantCalls[i])
				}
				if len(c) != 6 {
					t.Errorf("op code %q not 6 chars", c)
				}
			}
			if n := strings.Count(out.String(), goodbye); n != 1 {
				t.Errorf("goodbye printed %d times", n)
			}
		})
	}
}

func TestRunSharesReaderWithOperator(t *testing.T) {
	ops := &fakeOps{read: true}
	var out strings.Builder
	if err := Run(context.Background(), strings.NewReader("2\n100\n4\n"), &out, ops); err != nil {
		t.Fatal(err)
	}
	if len(ops.calls) != 1 || ops.calls[0] != "CREDIT" {
		t.Errorf("calls = %q", ops.calls)
	}
	if out.String() != menuText+menuText+goodbye {
		t.Errorf("output = %q", out.String())
	}
}

type errWriter struct{ failAfter, n int }

func (w *errWriter) Write(p []byte) (int, error) {
	if w.n >= w.failAfter {
		return 0, errors.New("write failed")
	}
	w.n++
	return len(p), nil
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestRunErrors(t *testing.T) {
	opErr := errors.New("operations unresolved")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name    string
		ctx     context.Context
		in      io.Reader
		out     io.Writer
		ops     Operator
		wantIs  error
		wantSub string
	}{
		{"nil operator", context.Background(), strings.NewReader("4\n"), io.Discard, nil, nil, "nil Operator"},
		{"cancelled ctx", cancelled, strings.NewReader("4\n"), io.Discard, &fakeOps{}, context.Canceled, ""},
		{"operation error", context.Background(), strings.NewReader("1\n4\n"), io.Discard, &fakeOps{err: opErr}, opErr, "choice 1"},
		{"credit error", context.Background(), strings.NewReader("2\n4\n"), io.Discard, &fakeOps{err: opErr}, opErr, "choice 2"},
		{"debit error", context.Background(), strings.NewReader("3\n4\n"), io.Discard, &fakeOps{err: opErr}, opErr, "choice 3"},
		{"read error", context.Background(), errReader{}, io.Discard, &fakeOps{}, nil, "read choice"},
		{"menu write error", context.Background(), strings.NewReader("4\n"), &errWriter{failAfter: 0}, &fakeOps{}, nil, "write menu"},
		{"goodbye write error", context.Background(), strings.NewReader("4\n"), &errWriter{failAfter: len(menuLines)}, &fakeOps{}, nil, "write goodbye"},
		{"invalid write error", context.Background(), strings.NewReader("8\n"), &errWriter{failAfter: len(menuLines)}, &fakeOps{}, nil, "choice 8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run(tt.ctx, tt.in, tt.out, tt.ops)
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("err = %v, want Is %v", err, tt.wantIs)
			}
			if tt.wantSub != "" && !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("err = %v, want containing %q", err, tt.wantSub)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if ChoiceViewBalance != 1 || ChoiceCredit != 2 || ChoiceDebit != 3 || ChoiceExit != 4 {
		t.Error("choice constants wrong")
	}
	for _, c := range []string{opTotal, opCredit, opDebit} {
		if len(c) != 6 {
			t.Errorf("op %q length %d", c, len(c))
		}
	}
	if opTotal != "TOTAL " || opCredit != "CREDIT" || opDebit != "DEBIT " {
		t.Error("op codes wrong")
	}
}