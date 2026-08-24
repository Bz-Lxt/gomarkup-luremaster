package hydro

import (
	"testing"
	"time"
)

func TestHarmonicDeterministicAndClassifies(t *testing.T) {
	q := Query{Lat: 30.38, Lon: 120.69, Tidal: true}
	from := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	a, err := HarmonicTide{}.Series(q, from, to)
	if err != nil || len(a) < 20 {
		t.Fatalf("series %d %v", len(a), err)
	}
	b, _ := HarmonicTide{}.Series(q, from, to)
	if a[0] != b[0] || a[len(a)/2] != b[len(b)/2] {
		t.Fatal("harmonic not deterministic")
	}
	seen := map[string]bool{}
	for _, p := range a {
		_, _, w := ClassifyTide(a, p.At)
		seen[w] = true
	}
	if len(seen) < 3 {
		t.Fatalf("expected diverse windows, got %v", seen)
	}
}

func TestTideRejectsInvertedWindow(t *testing.T) {
	to := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := HarmonicTide{}.Series(Query{}, to, to.Add(-time.Hour))
	if ClassifyError(err) != ClassValidation {
		t.Fatalf("class %s", ClassifyError(err))
	}
}

func TestInterpolateTideSlope(t *testing.T) {
	t0 := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	series := []TidePoint{{At: t0, HeightM: 0}, {At: t0.Add(time.Hour), HeightM: 1}}
	h, slope := interpTide(series, t0.Add(30*time.Minute))
	if h < 0.49 || h > 0.51 {
		t.Fatalf("h %v", h)
	}
	if slope < 0.9 || slope > 1.1 {
		t.Fatalf("slope %v", slope)
	}
}
