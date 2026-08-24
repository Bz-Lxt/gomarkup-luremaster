package hydro

import (
	"math"
	"time"
)

const (
	TideSlackLow   = "SLACK_LOW"
	TideEarlyFlood = "EARLY_FLOOD"
	TideThird      = "THIRD"
	TideHalf       = "HALF"
	TideSeventh    = "SEVENTH"
	TideSlackHigh  = "SLACK_HIGH"
	TideEarlyEbb   = "EARLY_EBB"
	TideRapidEbb   = "RAPID_EBB"
)

type HarmonicTide struct{}

type constituent struct {
	amp, speed, phase float64
}

// Simplified East China Sea mix (M2/S2/K1/O1). Deterministic, not a mock.
func (HarmonicTide) Series(q Query, from, to time.Time) ([]TidePoint, error) {
	if to.Before(from) {
		return nil, ClassifiedError{Class: ClassValidation, Message: "tide window inverted"}
	}
	latFactor := 0.65 + 0.35*math.Cos(q.Lat*math.Pi/180)
	cons := []constituent{
		{amp: 1.15 * latFactor, speed: 28.984104, phase: 0.35 + q.Lon*0.004},
		{amp: 0.42 * latFactor, speed: 30.000000, phase: 1.10 + q.Lon*0.003},
		{amp: 0.28 * latFactor, speed: 15.041069, phase: 2.20},
		{amp: 0.22 * latFactor, speed: 13.943035, phase: 0.80},
	}
	var out []TidePoint
	t0 := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	for t := from.UTC().Truncate(20 * time.Minute); !t.After(to.UTC()); t = t.Add(20 * time.Minute) {
		hours := t.Sub(t0).Hours()
		var h float64
		for _, c := range cons {
			h += c.amp * math.Cos((c.speed*math.Pi/180)*hours+c.phase)
		}
		out = append(out, TidePoint{At: t, HeightM: math.Round(h*1000) / 1000})
	}
	return out, nil
}

func ClassifyTide(series []TidePoint, at time.Time) (height, phasePct float64, window string) {
	if len(series) == 0 {
		return 0, 50, TideHalf
	}
	h, slope := interpTide(series, at)
	minH, maxH := series[0].HeightM, series[0].HeightM
	for _, p := range series {
		if p.HeightM < minH {
			minH = p.HeightM
		}
		if p.HeightM > maxH {
			maxH = p.HeightM
		}
	}
	span := maxH - minH
	if span < 1e-6 {
		return h, 50, TideHalf
	}
	phasePct = (h - minH) / span * 100
	rising := slope > 0.008
	falling := slope < -0.008
	switch {
	case phasePct <= 8 && math.Abs(slope) < 0.012:
		window = TideSlackLow
	case phasePct >= 92 && math.Abs(slope) < 0.012:
		window = TideSlackHigh
	case rising && phasePct < 28:
		window = TideEarlyFlood
	case rising && phasePct < 42:
		window = TideThird
	case rising && phasePct < 58:
		window = TideHalf
	case rising && phasePct < 82:
		window = TideSeventh
	case falling && phasePct > 75:
		window = TideEarlyEbb
	case falling:
		window = TideRapidEbb
	default:
		window = TideHalf
	}
	return h, math.Round(phasePct*10) / 10, window
}

func interpTide(series []TidePoint, at time.Time) (height, slope float64) {
	if at.Before(series[0].At) {
		return series[0].HeightM, 0
	}
	last := series[len(series)-1]
	if !at.Before(last.At) {
		return last.HeightM, 0
	}
	for i := 1; i < len(series); i++ {
		a, b := series[i-1], series[i]
		if at.Before(b.At) || at.Equal(b.At) {
			span := b.At.Sub(a.At).Hours()
			if span <= 0 {
				return a.HeightM, 0
			}
			r := at.Sub(a.At).Hours() / span
			h := a.HeightM + (b.HeightM-a.HeightM)*r
			return h, b.HeightM - a.HeightM
		}
	}
	return last.HeightM, 0
}
