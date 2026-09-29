package services

import (
	"testing"
	"time"
)

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		input string
		want  string
		ok    bool
	}{
		{"13800138000", "13800138000", true},
		{" 138-0013-8000 ", "13800138000", true},
		{"+8613800138000", "13800138000", true},
		{"1234567", "1234567", true},
		{"abc", "", false},
		{"13800", "", false},
		{"", "", false},
		{"1234567890123456", "", false},
	}

	for _, tc := range cases {
		got, err := NormalizePhone(tc.input)
		if tc.ok {
			if err != nil {
				t.Fatalf("expected %q to normalize, got error %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("expected %q to normalize to %q, got %q", tc.input, tc.want, got)
			}
		} else if err == nil {
			t.Fatalf("expected %q to be rejected, got %q", tc.input, got)
		}
	}
}

func storedEntry(phone string) (codeEntry, bool) {
	codeStoreMu.Lock()
	defer codeStoreMu.Unlock()
	e, ok := codeStore[phone]
	return e, ok
}

func TestSendSignInCodeStoresSixDigitCode(t *testing.T) {
	phone := "13800000001"

	if _, err := SendSignInCode(phone); err != nil {
		t.Fatalf("send code: %v", err)
	}

	entry, ok := storedEntry(phone)
	if !ok {
		t.Fatal("expected a stored code entry")
	}
	if len(entry.code) != 6 {
		t.Fatalf("expected a 6-digit code, got %q", entry.code)
	}
	if !time.Now().Before(entry.expiresAt) {
		t.Fatal("expected the code to be unexpired")
	}
}

func TestConsumeSignInCode(t *testing.T) {
	phone := "13800000002"

	if _, err := SendSignInCode(phone); err != nil {
		t.Fatalf("send code: %v", err)
	}
	codeStoreMu.Lock()
	entry := codeStore[phone]
	codeStoreMu.Unlock()

	// Codes are single-use: even a wrong attempt consumes the stored code,
	// which blocks brute forcing.
	if err := ConsumeSignInCode(phone, "000000"); err == nil {
		t.Fatal("expected a wrong code to be rejected")
	}
	if _, ok := storedEntry(phone); ok {
		t.Fatal("expected a wrong attempt to consume the code")
	}

	// A fresh code verifies once.
	if _, err := SendSignInCode(phone); err != nil {
		t.Fatalf("resend code: %v", err)
	}
	codeStoreMu.Lock()
	entry = codeStore[phone]
	codeStoreMu.Unlock()

	if err := ConsumeSignInCode(phone, entry.code); err != nil {
		t.Fatalf("expected the correct code to verify: %v", err)
	}
	if err := ConsumeSignInCode(phone, entry.code); err == nil {
		t.Fatal("expected the code to be single-use")
	}

	// Expired codes are rejected.
	codeStoreMu.Lock()
	codeStore[phone] = codeEntry{code: "123456", expiresAt: time.Now().Add(-time.Minute)}
	codeStoreMu.Unlock()
	if err := ConsumeSignInCode(phone, "123456"); err == nil {
		t.Fatal("expected an expired code to be rejected")
	}

	// Blank codes get their own message before any store lookup.
	if _, err := SendSignInCode(phone); err != nil {
		t.Fatalf("send code: %v", err)
	}
	if err := ConsumeSignInCode(phone, "   "); err != ErrCodeRequired {
		t.Fatalf("expected ErrCodeRequired for a blank code, got %v", err)
	}
}
