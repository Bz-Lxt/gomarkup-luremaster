package hydro

import (
	"math"
	"time"
)

const (
	MoonNew      = "NEW"
	MoonWaxCres  = "WAXING_CRESCENT"
	MoonFirst    = "FIRST_QUARTER"
	MoonWaxGib   = "WAXING_GIBBOUS"
	MoonFull     = "FULL"
	MoonWanGib   = "WANING_GIBBOUS"
	MoonLast     = "LAST_QUARTER"
	MoonWanCres  = "WANING_CRESCENT"
)

func MoonAt(t time.Time) (phase string, illum float64) {
	jd := julianDate(t.UTC())
	// Synodic cycle referenced to known new moon 2000-01-06 18:14 UTC
	days := jd - 2451550.1
	age := math.Mod(days, 29.530588853)
	if age < 0 {
		age += 29.530588853
	}
	illum = (1 - math.Cos(2*math.Pi*age/29.530588853)) / 2 * 100
	switch {
	case age < 1.84566:
		phase = MoonNew
	case age < 5.53699:
		phase = MoonWaxCres
	case age < 9.22831:
		phase = MoonFirst
	case age < 12.91963:
		phase = MoonWaxGib
	case age < 16.61096:
		phase = MoonFull
	case age < 20.30228:
		phase = MoonWanGib
	case age < 23.99361:
		phase = MoonLast
	case age < 27.68493:
		phase = MoonWanCres
	default:
		phase = MoonNew
	}
	return phase, math.Round(illum*10) / 10
}

func julianDate(t time.Time) float64 {
	y := t.Year()
	m := int(t.Month())
	d := float64(t.Day()) + float64(t.Hour())/24 + float64(t.Minute())/1440 + float64(t.Second())/86400
	if m <= 2 {
		y--
		m += 12
	}
	a := math.Floor(float64(y) / 100)
	b := 2 - a + math.Floor(a/4)
	return math.Floor(365.25*float64(y+4716)) + math.Floor(30.6001*float64(m+1)) + d + b - 1524.5
}
