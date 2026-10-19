package store

import (
	"context"
	"errors"
	"sync"
	"testing"

	"accountsystem/internal/money"
)

const (
	InitialBalanceCents = money.InitialBalance
	MaxBalanceCents     = money.Max
)

type fakeStore struct {
	readVal  money.Amount
	readErr  error
	writeErr error
	written  []money.Amount
	reads    int
}

func (f *fakeStore) Read(ctx context.Context) (money.Amount, error) {
	f.reads++
	return f.readVal, f.readErr
}

func (f *fakeStore) Write(ctx context.Context, cents money.Amount) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.written = append(f.written, cents)
	return nil
}

func TestNewMemoryStoreInitialBalance(t *testing.T) {
	s := NewMemoryStore(money.InitialBalance)
	got, err := s.Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != InitialBalanceCents {
		t.Fatalf("got %d, want %d", got, InitialBalanceCents)
	}
	if got != 100000 {
		t.Fatalf("got %d, want 100000", got)
	}
}

type step struct {
	op      string
	in      money.Amount
	want    money.Amount
	wantErr error
}

func TestCallSequences(t *testing.T) {
	tests := []struct {
		name       string
		steps      []step
		wantStored money.Amount
	}{
		{"first READ returns 1000.00", []step{{op: "READ", in: 0, want: 100000}}, 100000},
		{"WRITE 1250.50 leaves caller unchanged", []step{{op: "WRITE", in: 125050, want: 125050}}, 125050},
		{"WRITE then READ persists", []step{
			{op: "WRITE", in: 125050, want: 125050},
			{op: "READ", in: 0, want: 125050},
		}, 125050},
		{"multiple WRITEs last wins", []step{
			{op: "WRITE", in: 50000, want: 50000},
			{op: "WRITE", in: 75025, want: 75025},
			{op: "READ", in: 0, want: 75025},
		}, 75025},
		{"repeated READs same value", []step{
			{op: "READ", in: 1, want: 100000},
			{op: "READ", in: 2, want: 100000},
			{op: "READ", in: 3, want: 100000},
		}, 100000},
		{"WRITE zero then READ", []step{
			{op: "WRITE", in: 0, want: 0},
			{op: "READ", in: 5, want: 0},
		}, 0},
		{"WRITE max then READ", []step{
			{op: "WRITE", in: MaxBalanceCents, want: 99999999},
			{op: "READ", in: 0, want: 99999999},
		}, 99999999},
		{"padded codes", []step{
			{op: "WRITE ", in: 1234, want: 1234},
			{op: "READ  ", in: 0, want: 1234},
		}, 1234},
		{"codes truncated to six chars", []step{
			{op: "WRITE XYZ", in: 4321, want: 4321},
			{op: "READ  extra", in: 0, want: 4321},
		}, 4321},
		{"DELETE is no-op", []step{{op: "DELETE", in: 777, want: 777}}, 100000},
		{"lowercase read is no-op", []step{{op: "read", in: 777, want: 777}}, 100000},
		{"lowercase write is no-op", []step{{op: "write", in: 777, want: 777}}, 100000},
		{"all spaces is no-op", []step{{op: "      ", in: 777, want: 777}}, 100000},
		{"empty is no-op", []step{{op: "", in: 777, want: 777}}, 100000},
		{"leading space READ is no-op", []step{{op: " READ ", in: 777, want: 777}}, 100000},
		{"leading space WRITE is no-op", []step{{op: " WRITE", in: 777, want: 777}}, 100000},
		{"WRITE over max rejected", []step{{op: "WRITE", in: MaxBalanceCents + 1, want: MaxBalanceCents + 1, wantErr: ErrOutOfRange}}, 100000},
		{"WRITE negative rejected", []step{{op: "WRITE", in: -1, want: -1, wantErr: ErrOutOfRange}}, 100000},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s := NewMemoryStore(money.InitialBalance)
			for i, st := range tc.steps {
				bal := st.in
				err := Call(ctx, s, st.op, &bal)
				if st.wantErr != nil {
					if !errors.Is(err, st.wantErr) {
						t.Fatalf("step %d: err=%v, want %v", i, err, st.wantErr)
					}
				} else if err != nil {
					t.Fatalf("step %d: unexpected error: %v", i, err)
				}
				if bal != st.want {
					t.Fatalf("step %d (%q): balance=%d, want %d", i, st.op, bal, st.want)
				}
			}
			got, err := s.Read(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.wantStored {
				t.Fatalf("stored=%d, want %d", got, tc.wantStored)
			}
		})
	}
}

func TestCallWithFakeStore(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name       string
		store      *fakeStore
		op         string
		in         money.Amount
		want       money.Amount
		wantErr    error
		wantReads  int
		wantWrites int
	}{
		{"read success", &fakeStore{readVal: 42}, "READ", 0, 42, nil, 1, 0},
		{"read error keeps balance", &fakeStore{readErr: boom}, "READ", 9, 9, boom, 1, 0},
		{"write success", &fakeStore{}, "WRITE", 55, 55, nil, 0, 1},
		{"write error", &fakeStore{writeErr: boom}, "WRITE", 55, 55, boom, 0, 0},
		{"unknown op touches nothing", &fakeStore{}, "DELETE", 3, 3, nil, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bal := tc.in
			err := Call(context.Background(), tc.store, tc.op, &bal)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err=%v, want %v", err, tc.wantErr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if bal != tc.want {
				t.Fatalf("balance=%d, want %d", bal, tc.want)
			}
			if tc.store.reads != tc.wantReads || len(tc.store.written) != tc.wantWrites {
				t.Fatalf("reads=%d writes=%d", tc.store.reads, len(tc.store.written))
			}
		})
	}
}

func TestCallNilArguments(t *testing.T) {
	bal := money.Amount(1)
	if err := Call(context.Background(), nil, "READ", &bal); err == nil {
		t.Fatal("expected error for nil store")
	}
	if err := Call(context.Background(), NewMemoryStore(money.InitialBalance), "READ", nil); err == nil {
		t.Fatal("expected error for nil balance")
	}
}

func TestMemoryStoreWriteRange(t *testing.T) {
	tests := []struct {
		name    string
		cents   money.Amount
		wantErr bool
		want    money.Amount
	}{
		{"zero", 0, false, 0},
		{"max", MaxBalanceCents, false, MaxBalanceCents},
		{"mid", 125050, false, 125050},
		{"negative", -1, true, InitialBalanceCents},
		{"over max", MaxBalanceCents + 1, true, InitialBalanceCents},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewMemoryStore(money.InitialBalance)
			err := s.Write(context.Background(), tc.cents)
			if tc.wantErr != errors.Is(err, ErrOutOfRange) {
				t.Fatalf("err=%v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got, _ := s.Read(context.Background())
			if got != tc.want {
				t.Fatalf("stored=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestMemoryStoreCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := NewMemoryStore(money.InitialBalance)
	if _, err := s.Read(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("read err=%v", err)
	}
	if err := s.Write(ctx, 5); !errors.Is(err, context.Canceled) {
		t.Fatalf("write err=%v", err)
	}
	got, _ := s.Read(context.Background())
	if got != InitialBalanceCents {
		t.Fatalf("stored=%d", got)
	}
	bal := money.Amount(7)
	if err := Call(ctx, s, "READ", &bal); !errors.Is(err, context.Canceled) || bal != 7 {
		t.Fatalf("call err=%v bal=%d", err, bal)
	}
}

func TestMemoryStoreConcurrent(t *testing.T) {
	s := NewMemoryStore(money.InitialBalance)
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(v money.Amount) {
			defer wg.Done()
			if err := s.Write(ctx, v); err != nil {
				t.Error(err)
			}
		}(money.Amount(i))
		go func() {
			defer wg.Done()
			v, err := s.Read(ctx)
			if err != nil || v < 0 || v > MaxBalanceCents {
				t.Errorf("v=%d err=%v", v, err)
			}
		}()
	}
	wg.Wait()
}
