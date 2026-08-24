package hydro_test

import (
	"testing"
	"time"

	"luremaster/internal/hydro"
)

type pressureDawnWeather struct {
	at time.Time
}

func (w pressureDawnWeather) Hourly(_ hydro.Query, _, _ time.Time) ([]hydro.Sample, error) {
	return []hydro.Sample{
		{At: w.at.Add(-3 * time.Hour), PressureHPa: 1012, TempC: 30, WindDirDeg: 90, WindMS: 0.1},
		{At: w.at, PressureHPa: 1008, TempC: 30, WindDirDeg: 90, WindMS: 0.1},
	}, nil
}

func TestRetroAttributesPressureDawnSynergyToPeriod(t *testing.T) {
	at := time.Date(2026, 8, 24, 21, 30, 0, 0, time.UTC)
	waterTemp := 30.0
	engine := hydro.NewEngine(pressureDawnWeather{at: at}, nil)

	snapshot, err := engine.Retro(hydro.Query{
		Lat:          30.25,
		Lon:          122.18,
		At:           at,
		ShoreBearing: 0,
		Tidal:        false,
		WaterTempC:   &waterTemp,
	})
	if err != nil {
		t.Fatal(err)
	}

	var pressure, period *hydro.Contribution
	for i := range snapshot.Contributions {
		switch snapshot.Contributions[i].Node {
		case "pressure":
			pressure = &snapshot.Contributions[i]
		case "period":
			period = &snapshot.Contributions[i]
		}
	}
	if pressure == nil || period == nil {
		t.Fatalf("missing public contributions: %+v", snapshot.Contributions)
	}
	if pressure.Bonus != 0 {
		t.Fatalf("pressure bonus = %.1f, want 0", pressure.Bonus)
	}
	if period.Bonus != 8 {
		t.Fatalf("period bonus = %.1f, want 8", period.Bonus)
	}
}
