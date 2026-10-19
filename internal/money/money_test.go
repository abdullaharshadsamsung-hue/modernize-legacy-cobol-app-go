package money

import "testing"

func TestString(t *testing.T) {
	tests := []struct {
		name string
		in   Amount
		want string
	}{
		{"zero", 0, "000000.00"},
		{"initial", InitialBalance, "001000.00"},
		{"cents", 5, "000000.05"},
		{"max", Max, "999999.99"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValid(t *testing.T) {
	tests := []struct {
		in   Amount
		want bool
	}{
		{0, true},
		{Max, true},
		{-1, false},
		{Max + 1, false},
	}
	for _, tt := range tests {
		if got := tt.in.Valid(); got != tt.want {
			t.Errorf("Amount(%d).Valid() = %v, want %v", int64(tt.in), got, tt.want)
		}
	}
}

func TestAddSub(t *testing.T) {
	tests := []struct {
		name string
		got  Amount
		want Amount
	}{
		{"add simple", InitialBalance.Add(25050), 125050},
		{"add overflow truncates", Max.Add(100), 99},
		{"sub simple", InitialBalance.Sub(10000), 90000},
		{"sub to zero", InitialBalance.Sub(InitialBalance), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %d, want %d", int64(tt.got), int64(tt.want))
			}
		})
	}
}

func TestParseAccept(t *testing.T) {
	tests := []struct {
		in   string
		want Amount
	}{
		{"100", 10000},
		{"100\n", 10000},
		{" 7.5 \n", 750},
		{"12.345", 1234},
		{".5", 50},
		{"-5", 500},
		{"+5", 500},
		{"1234567", 23456700},
		{"abc", 0},
		{"", 0},
		{"1.2.3", 0},
	}
	for _, tt := range tests {
		if got := ParseAccept(tt.in); got != tt.want {
			t.Errorf("ParseAccept(%q) = %d, want %d", tt.in, int64(got), int64(tt.want))
		}
	}
}

func TestParseChoice(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"1\n", 1},
		{" 4 ", 4},
		{"12", 2},
		{"x", 0},
		{"", 0},
		{"-1", 0},
	}
	for _, tt := range tests {
		if got := ParseChoice(tt.in); got != tt.want {
			t.Errorf("ParseChoice(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
