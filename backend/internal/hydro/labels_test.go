package hydro

import "testing"

func TestLabelsCoverEnums(t *testing.T) {
	for _, c := range []string{TrendCrashDown, TrendFall, TrendStable, TrendRise, TrendCrashUp} {
		if LabelOf("pressure", c) == c {
			t.Fatalf("missing pressure label %s", c)
		}
	}
	for _, c := range []string{TideThird, TideSlackLow, TideSlackHigh, TideEarlyFlood} {
		if LabelOf("tide", c) == c {
			t.Fatalf("missing tide label %s", c)
		}
	}
	if LabelOf("moon", MoonFull) != "满月" {
		t.Fatal(LabelOf("moon", MoonFull))
	}
}
