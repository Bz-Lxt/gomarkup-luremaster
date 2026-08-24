package hydro

import "math"

var compass = []string{
	"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE",
	"S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW",
}

func WindLabel(deg float64) string {
	for deg < 0 {
		deg += 360
	}
	idx := int(math.Mod(deg+11.25, 360) / 22.5)
	if idx < 0 || idx >= len(compass) {
		return "N"
	}
	return compass[idx]
}

func Beaufort(ms float64) int {
	thresholds := []float64{0.3, 1.6, 3.4, 5.5, 8.0, 10.8, 13.9, 17.2, 20.8, 24.5, 28.5, 32.7}
	for i, t := range thresholds {
		if ms < t {
			return i
		}
	}
	return 12
}

const (
	AspectOnshore  = "ONSHORE"
	AspectOffshore = "OFFSHORE"
	AspectCross    = "CROSS"
)

func ShoreAspect(windDeg, shoreBearing float64) string {
	diff := math.Abs(math.Mod(windDeg-shoreBearing+540, 360) - 180)
	if diff <= 45 {
		return AspectOnshore
	}
	if diff >= 135 {
		return AspectOffshore
	}
	return AspectCross
}
