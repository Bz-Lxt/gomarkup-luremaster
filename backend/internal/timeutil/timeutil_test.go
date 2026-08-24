package timeutil

import (
	"testing"
	"time"
)

func TestParseCatchLocalBeijing(t *testing.T) {
	utc, loc, err := ParseCatchLocal("2026-08-24 14:15", "Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	if loc.String() != "Asia/Shanghai" && loc != Beijing {
		t.Fatalf("loc %s", loc)
	}
	if utc.Hour() != 6 || utc.Minute() != 15 {
		t.Fatalf("expected 06:15 UTC, got %s", utc)
	}
}

func TestCivilDateUsesShanghai(t *testing.T) {
	// 2026-08-24 02:00 CST is still 23rd in UTC
	t0 := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	y, m, d := CivilDate(t0, Beijing)
	if y != 2026 || m != 8 || d != 24 {
		t.Fatalf("got %d-%d-%d", y, m, d)
	}
}
