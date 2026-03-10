package api

import (
	"testing"
	"time"
)

func TestParsePeriod(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)

	tests := map[string]time.Time{
		"5m": now.Add(-5 * time.Minute),
		"1h": now.Add(-time.Hour),
		"1D": now.AddDate(0, 0, -1),
		"1W": now.AddDate(0, 0, -7),
		"1M": now.AddDate(0, -1, 0),
	}

	for period, want := range tests {
		got, err := parsePeriod(period, now)
		if err != nil {
			t.Fatalf("parsePeriod(%q) returned error: %v", period, err)
		}
		if !got.Equal(want) {
			t.Fatalf("parsePeriod(%q): expected %s, got %s", period, want, got)
		}
	}
}

func TestParsePeriodRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	if _, err := parsePeriod("", time.Now()); err == nil {
		t.Fatal("expected error for empty period")
	}
	if _, err := parsePeriod("abc", time.Now()); err == nil {
		t.Fatal("expected error for invalid period")
	}
}
