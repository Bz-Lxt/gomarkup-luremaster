package hydro

import (
	"math"
	"time"
)

const (
	TrendCrashDown = "CRASH_DOWN"
	TrendFall      = "FALL"
	TrendStable    = "STABLE"
	TrendRise      = "RISE"
	TrendCrashUp   = "CRASH_UP"
)

func ClassifyPressure(delta3h float64) string {
	switch {
	case delta3h <= -3.0:
		return TrendCrashDown
	case delta3h >= 3.0:
		return TrendCrashUp
	case math.Abs(delta3h) < 0.5:
		return TrendStable
	case delta3h < 0:
		return TrendFall
	default:
		return TrendRise
	}
}

func PressureDelta3h(hourly []Sample, at time.Time) float64 {
	now := Interpolate(hourly, at)
	past := Interpolate(hourly, at.Add(-3*timeHour))
	if now == nil || past == nil {
		return 0
	}
	return now.PressureHPa - past.PressureHPa
}

const timeHour = 1e9 * 60 * 60

func Interpolate(hourly []Sample, at time.Time) *Sample {
	if len(hourly) == 0 {
		return nil
	}
	if at.Before(hourly[0].At) {
		s := hourly[0]
		return &s
	}
	last := hourly[len(hourly)-1]
	if !at.Before(last.At) {
		s := last
		return &s
	}
	for i := 1; i < len(hourly); i++ {
		a, b := hourly[i-1], hourly[i]
		if at.Equal(b.At) {
			s := b
			return &s
		}
		if at.Before(b.At) {
			span := b.At.Sub(a.At).Seconds()
			if span <= 0 {
				s := a
				return &s
			}
			r := at.Sub(a.At).Seconds() / span
			return &Sample{
				At:          at,
				PressureHPa: lerp(a.PressureHPa, b.PressureHPa, r),
				TempC:       lerp(a.TempC, b.TempC, r),
				WindDirDeg:  lerpAngle(a.WindDirDeg, b.WindDirDeg, r),
				WindMS:      lerp(a.WindMS, b.WindMS, r),
			}
		}
	}
	s := last
	return &s
}

func lerp(a, b, r float64) float64 { return a + (b-a)*r }

func lerpAngle(a, b, r float64) float64 {
	d := math.Mod(b-a+540, 360) - 180
	return math.Mod(a+d*r+360, 360)
}
