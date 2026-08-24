package hydro

import (
	"math"
	"testing"
	"time"
)

func TestInterpolateBindErrorUnderOneMinute(t *testing.T) {
	base := time.Date(2026, 8, 20, 6, 0, 0, 0, time.UTC)
	hourly := []Sample{
		{At: base, PressureHPa: 1010, TempC: 24, WindDirDeg: 90, WindMS: 3},
		{At: base.Add(time.Hour), PressureHPa: 1006, TempC: 26, WindDirDeg: 100, WindMS: 4},
	}
	at := base.Add(15 * time.Minute)
	got := Interpolate(hourly, at)
	if got == nil {
		t.Fatal("nil interp")
	}
	if got.At.Sub(at) != 0 {
		t.Fatalf("bind residual %s", got.At.Sub(at))
	}
	if math.Abs(got.PressureHPa-1009) > 0.05 {
		t.Fatalf("pressure %v", got.PressureHPa)
	}
}

func TestClassifyPressureThresholds(t *testing.T) {
	cases := []struct {
		d    float64
		want string
	}{
		{-3.1, TrendCrashDown}, {-0.8, TrendFall}, {0.2, TrendStable}, {1.2, TrendRise}, {3.2, TrendCrashUp},
	}
	for _, c := range cases {
		if g := ClassifyPressure(c.d); g != c.want {
			t.Fatalf("delta %v got %s want %s", c.d, g, c.want)
		}
	}
}

func TestMockDeterministic(t *testing.T) {
	q := Query{Lat: 30.25, Lon: 122.18, At: time.Date(2026, 8, 20, 6, 15, 0, 0, time.UTC)}
	a, err := MockWeather{}.Hourly(q, q.At.Add(-2*time.Hour), q.At.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := MockWeather{}.Hourly(q, q.At.Add(-2*time.Hour), q.At.Add(2*time.Hour))
	if len(a) != len(b) || a[0] != b[0] {
		t.Fatal("mock not deterministic")
	}
}

func TestEngineRetroFrenzyPath(t *testing.T) {
	at := time.Date(2026, 8, 24, 21, 15, 0, 0, time.UTC) // 05:15 Beijing = dawn
	eng := NewEngine(scriptedWeather{
		samples: []Sample{
			{At: at.Add(-4 * time.Hour), PressureHPa: 1016, TempC: 22, WindDirDeg: 90, WindMS: 3.2},
			{At: at.Add(-3 * time.Hour), PressureHPa: 1015, TempC: 22, WindDirDeg: 90, WindMS: 3.2},
			{At: at.Add(-1 * time.Hour), PressureHPa: 1010, TempC: 21, WindDirDeg: 88, WindMS: 3.4},
			{At: at, PressureHPa: 1008, TempC: 21, WindDirDeg: 90, WindMS: 3.5},
			{At: at.Add(time.Hour), PressureHPa: 1007, TempC: 21, WindDirDeg: 92, WindMS: 3.6},
		},
	}, HarmonicTide{})
	wt := 22.0
	snap, err := eng.Retro(Query{Lat: 30.25, Lon: 122.2, At: at, ShoreBearing: 90, Tidal: true, WaterTempC: &wt})
	if err != nil {
		t.Fatal(err)
	}
	if snap.PressureTrend != TrendCrashDown {
		t.Fatalf("trend %s", snap.PressureTrend)
	}
	if snap.BindErrorSec > 60 {
		t.Fatalf("bind error %d", snap.BindErrorSec)
	}
	if len(snap.Contributions) != 6 {
		t.Fatalf("contrib %d", len(snap.Contributions))
	}
	if snap.BiteScore < 0 || snap.BiteScore > 100 {
		t.Fatalf("score %v", snap.BiteScore)
	}
}

func TestRetryPolicy(t *testing.T) {
	if Retryable(ClassAuth) || Retryable(ClassValidation) || Retryable(ClassBounds) {
		t.Fatal("non-transient must not retry")
	}
	if !Retryable(ClassTransient) {
		t.Fatal("transient should retry")
	}
}

func TestMoonAndWind(t *testing.T) {
	phase, illum := MoonAt(time.Date(2026, 8, 28, 4, 0, 0, 0, time.UTC))
	if illum < 0 || illum > 100 {
		t.Fatalf("illum %v", illum)
	}
	if phase == "" {
		t.Fatal("empty phase")
	}
	if WindLabel(0) != "N" || WindLabel(90) != "E" {
		t.Fatal("compass")
	}
	if Beaufort(0.1) != 0 || Beaufort(3.3) != 2 {
		t.Fatalf("beaufort %d %d", Beaufort(0.1), Beaufort(3.3))
	}
}

type scriptedWeather struct{ samples []Sample }

func (s scriptedWeather) Hourly(_ Query, _, _ time.Time) ([]Sample, error) { return s.samples, nil }
