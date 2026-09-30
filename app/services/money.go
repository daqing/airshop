package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var pricePattern = regexp.MustCompile(`^(\d{1,10})(?:\.(\d{1,2}))?$`)

// ParsePriceToCents parses a display price like "12", "12.3" or "12.34" into
// minor units (cents) per the T0.5 convention. Negative or more precise
// values are rejected.
func ParsePriceToCents(input string) (int64, error) {
	s := strings.TrimSpace(input)
	m := pricePattern.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid price %q: use a non-negative amount with at most two decimals", input)
	}

	cents, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, err
	}
	cents *= 100

	if m[2] != "" {
		frac, err := strconv.ParseInt(m[2], 10, 64)
		if err != nil {
			return 0, err
		}
		if len(m[2]) == 1 {
			frac *= 10
		}
		cents += frac
	}
	return cents, nil
}

// FormatCents renders minor units as a decimal display string ("12.34").
func FormatCents(c int64) string {
	if c < 0 {
		return "-" + FormatCents(-c)
	}
	return fmt.Sprintf("%d.%02d", c/100, c%100)
}
