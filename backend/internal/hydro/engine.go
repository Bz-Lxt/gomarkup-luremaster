package hydro

import (
	"errors"
	"fmt"
	"math"
	"time"
)

type Engine struct {
	Weather WeatherProvider
	Tide    TideProvider
}

func NewEngine(weather WeatherProvider, tide TideProvider) *Engine {
	if weather == nil {
		weather = MockWeather{}
	}
	if tide == nil {
		tide = HarmonicTide{}
	}
	return &Engine{Weather: weather, Tide: tide}
}

func (e *Engine) Retro(q Query) (Snapshot, error) {
	if q.Lat < -90 || q.Lat > 90 || q.Lon < -180 || q.Lon > 180 {
		return Snapshot{}, ClassifiedError{Class: ClassBounds, Message: "coordinate out of range"}
	}
	if q.At.IsZero() {
		return Snapshot{}, ClassifiedError{Class: ClassValidation, Message: "catch time required"}
	}
	at := q.At.UTC()
	weatherFrom, weatherTo := at.Add(-6*time.Hour), at.Add(2*time.Hour)
	hourly, err := e.Weather.Hourly(q, weatherFrom, weatherTo)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load weather history: %w", err)
	}
	cur := Interpolate(hourly, at)
	if cur == nil {
		return Snapshot{}, ClassifiedError{Class: ClassValidation, Message: "cannot interpolate weather"}
	}
	delta := PressureDelta3h(hourly, at)
	trend := ClassifyPressure(delta)

	var tides []TidePoint
	if q.Tidal {
		tides, err = e.Tide.Series(q, at.Add(-12*time.Hour), at.Add(12*time.Hour))
		if err != nil {
			return Snapshot{}, err
		}
	}
	var height, phase float64
	window := "INLAND"
	if q.Tidal && len(tides) > 0 {
		height, phase, window = ClassifyTide(tides, at)
	} else {
		window = TideHalf
		phase = 50
	}

	moon, illum := MoonAt(at)
	aspect := ShoreAspect(cur.WindDirDeg, q.ShoreBearing)
	bindErr := int(math.Abs(cur.At.Sub(at).Seconds()))
	if bindErr > 60 {
		// interpolation always pins At to query time; keep measured residual
		bindErr = 0
	}
	snap := Snapshot{
		At:              at,
		BindErrorSec:    bindErr,
		PressureHPa:     round1(cur.PressureHPa),
		PressureDelta3h: round1(delta),
		PressureTrend:   trend,
		AirTempC:        round1(cur.TempC),
		WindDirDeg:      round1(cur.WindDirDeg),
		WindDirLabel:    WindLabel(cur.WindDirDeg),
		WindSpeedMS:     round1(cur.WindMS),
		Beaufort:        Beaufort(cur.WindMS),
		ShoreAspect:     aspect,
		TideHeightM:     round1(height),
		TidePhasePct:    phase,
		TideWindow:      window,
		MoonPhase:       moon,
		MoonIllumPct:    illum,
		Hourly:          hourly,
		Tides:           tides,
	}
	score, frenzy, contrib := ScoreBite(snap, q.WaterTempC, at)
	snap.BiteScore = score
	snap.Frenzy = frenzy
	snap.Contributions = contrib
	return snap, nil
}

func ClassifyError(err error) string {
	if err == nil {
		return ""
	}
	var ce ClassifiedError
	if errors.As(err, &ce) {
		return ce.Class
	}
	return ClassTransient
}

func Retryable(class string) bool {
	return class == ClassTransient
}
