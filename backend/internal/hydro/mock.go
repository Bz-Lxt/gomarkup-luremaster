package hydro

import (
	"hash/fnv"
	"math"
	"time"
)

type MockWeather struct{}

func (MockWeather) Hourly(_ Query, from, to time.Time) ([]Sample, error) {
	if to.Before(from) {
		return nil, ClassifiedError{Class: ClassValidation, Message: "weather window inverted"}
	}
	var out []Sample
	from = from.UTC().Truncate(time.Hour)
	to = to.UTC()
	for t := from; !t.After(to); t = t.Add(time.Hour) {
		out = append(out, sampleAt(t))
	}
	return out, nil
}

func sampleAt(t time.Time) Sample {
	seed := hashTime(t)
	hour := float64(t.Hour())
	// Diurnal pressure wave + seeded synoptic drift. Same (lat ignored, time-only) would
	// collide across spots; mix lon/lat via unix + seed already includes unix.
	base := 1012.4 + 4.8*math.Sin(2*math.Pi*(hour-10)/24)
	synoptic := 6.5 * math.Sin(2*math.Pi*float64(t.Unix()/3600)/72)
	noise := (unit(seed) - 0.5) * 1.2
	temp := 18 + 8*math.Sin(2*math.Pi*(hour-15)/24) + (unit(seed>>3)-0.5)*2
	windDir := math.Mod(120+40*math.Sin(float64(t.YearDay()))+(unit(seed>>7)-0.5)*30+360, 360)
	windMS := 1.8 + 3.4*unit(seed>>11) + 1.2*math.Abs(math.Sin(2*math.Pi*hour/24))
	return Sample{
		At:          t,
		PressureHPa: round1(base + synoptic + noise),
		TempC:       round1(temp),
		WindDirDeg:  round1(windDir),
		WindMS:      round1(windMS),
	}
}

func hashTime(t time.Time) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(t.UTC().Format(time.RFC3339)))
	return h.Sum64()
}

func HashCoordTime(lat, lon float64, t time.Time) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(t.UTC().Format(time.RFC3339)))
	_, _ = h.Write([]byte{byte(int(lat*100) % 251), byte(int(lon*100) % 251)})
	return h.Sum64()
}

func unit(x uint64) float64 {
	return float64(x%10000) / 10000
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
