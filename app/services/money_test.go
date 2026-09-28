package services

import (
	"testing"
)

func TestParsePriceToCents(t *testing.T) {
	cases := []struct {
		input string
		want  int64
		ok    bool
	}{
		{"12", 1200, true},
		{"12.3", 1230, true},
		{"12.34", 1234, true},
		{"0.01", 1, true},
		{" 12.34 ", 1234, true},
		{"12.345", 0, false},
		{"-1", 0, false},
		{"abc", 0, false},
		{"", 0, false},
		{"1,234", 0, false},
	}

	for _, tc := range cases {
		got, err := ParsePriceToCents(tc.input)
		if tc.ok {
			if err != nil {
				t.Fatalf("expected %q to parse, got error %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("expected %q to parse to %d cents, got %d", tc.input, tc.want, got)
			}
		} else if err == nil {
			t.Fatalf("expected %q to be rejected, got %d cents", tc.input, got)
		}
	}
}

func TestFormatCents(t *testing.T) {
	cases := map[int64]string{
		0:     "0.00",
		1:     "0.01",
		1234:  "12.34",
		1200:  "12.00",
		12345: "123.45",
	}

	for cents, want := range cases {
		if got := FormatCents(cents); got != want {
			t.Fatalf("expected %d cents to format as %q, got %q", cents, want, got)
		}
	}
}
