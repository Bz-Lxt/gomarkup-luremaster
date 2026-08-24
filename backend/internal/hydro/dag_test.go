package hydro

import (
	"testing"
	"time"
)

func TestScoreBiteRangeAndExplainability(t *testing.T) {
	at := time.Date(2026, 8, 24, 21, 30, 0, 0, time.UTC)
	snap := Snapshot{
		PressureTrend: TrendCrashDown,
		TideWindow:    TideThird,
		ShoreAspect:   AspectOnshore,
		MoonPhase:     MoonFull,
		Beaufort:      3,
		AirTempC:      22,
	}
	wt := 22.0
	score, frenzy, contrib := ScoreBite(snap, &wt, at)
	if score < 70 {
		t.Fatalf("expected strong window, got %v", score)
	}
	if !frenzy && score >= 75 {
		t.Fatal("frenzy flag")
	}
	if len(contrib) != 6 {
		t.Fatalf("need 6 nodes, got %d", len(contrib))
	}
	for _, c := range contrib {
		if c.Reason == "" || c.Label == "" {
			t.Fatalf("empty explain %v", c)
		}
	}
}

func TestCrashUpHotWaterPenalized(t *testing.T) {
	at := time.Date(2026, 8, 24, 4, 0, 0, 0, time.UTC) // midday Beijing
	hi := 30.0
	high, _, _ := ScoreBite(Snapshot{PressureTrend: TrendCrashUp, TideWindow: TideSlackHigh, ShoreAspect: AspectOffshore, MoonPhase: MoonWanCres, Beaufort: 7, AirTempC: 32}, &hi, at)
	lo := 20.0
	good, _, _ := ScoreBite(Snapshot{PressureTrend: TrendCrashDown, TideWindow: TideThird, ShoreAspect: AspectOnshore, MoonPhase: MoonNew, Beaufort: 3, AirTempC: 22}, &lo, time.Date(2026, 8, 24, 21, 0, 0, 0, time.UTC))
	if high >= good {
		t.Fatalf("penalty failed %v >= %v", high, good)
	}
}
