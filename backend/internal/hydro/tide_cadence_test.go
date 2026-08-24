package hydro_test

import (
	"testing"
	"time"

	"luremaster/internal/hydro"
)

func sampledRise(base time.Time, step time.Duration) []hydro.TidePoint {
	points := []hydro.TidePoint{{At: base, HeightM: 0}}
	start := base.Add(5 * time.Hour)
	for elapsed := time.Duration(0); elapsed <= time.Hour; elapsed += step {
		points = append(points, hydro.TidePoint{
			At:      start.Add(elapsed),
			HeightM: 0.34 + 0.02*elapsed.Hours(),
		})
	}
	return append(points, hydro.TidePoint{At: base.Add(12 * time.Hour), HeightM: 1})
}

func TestClassifyTideIndependentOfSamplingCadence(t *testing.T) {
	base := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	at := base.Add(5*time.Hour + 30*time.Minute)

	_, hourlyPhase, hourlyWindow := hydro.ClassifyTide(sampledRise(base, time.Hour), at)
	_, tenMinutePhase, tenMinuteWindow := hydro.ClassifyTide(sampledRise(base, 10*time.Minute), at)

	if hourlyPhase != 35 || tenMinutePhase != 35 {
		t.Fatalf("phase changed with cadence: hourly=%v ten-minute=%v", hourlyPhase, tenMinutePhase)
	}
	if hourlyWindow != hydro.TideThird || tenMinuteWindow != hydro.TideThird {
		t.Fatalf("same rising tide classified differently: hourly=%s ten-minute=%s", hourlyWindow, tenMinuteWindow)
	}
}
