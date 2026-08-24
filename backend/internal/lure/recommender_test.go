package lure

import "testing"

func TestColdStartAtLeastThree(t *testing.T) {
	got := Recommend("YELLOWCHECK", HydroHint{PressureTrend: "CRASH_DOWN", TideWindow: "THIRD", WaterTempC: 22, WindBeaufort: 3}, nil)
	if len(got) < 3 {
		t.Fatalf("cold start %d", len(got))
	}
	for _, a := range got {
		if a.Reason == "" || a.LureType == "" {
			t.Fatalf("incomplete %+v", a)
		}
	}
}

func TestHistoryShiftsRanking(t *testing.T) {
	h := HydroHint{PressureTrend: "STABLE", TideWindow: "HALF", WaterTempC: 20, WindBeaufort: 2}
	hist := []HistoryHit{}
	for i := 0; i < 12; i++ {
		hist = append(hist, HistoryHit{LureType: "SOFT", Caught: true})
	}
	got := Recommend("MANDARIN", h, hist)
	if got[0].LureType != "SOFT" {
		t.Fatalf("expected SOFT first, got %s", got[0].LureType)
	}
}
