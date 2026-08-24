package hydro

import "time"

type Sample struct {
	At          time.Time
	PressureHPa float64
	TempC       float64
	WindDirDeg  float64
	WindMS      float64
}

type TidePoint struct {
	At       time.Time
	HeightM  float64
}

type Snapshot struct {
	At              time.Time
	BindErrorSec    int
	PressureHPa     float64
	PressureDelta3h float64
	PressureTrend   string
	AirTempC        float64
	WindDirDeg      float64
	WindDirLabel    string
	WindSpeedMS     float64
	Beaufort        int
	ShoreAspect     string
	TideHeightM     float64
	TidePhasePct    float64
	TideWindow      string
	MoonPhase       string
	MoonIllumPct    float64
	BiteScore       float64
	Frenzy          bool
	Contributions   []Contribution
	Hourly          []Sample
	Tides           []TidePoint
}

type Contribution struct {
	Node    string  `json:"node"`
	Label   string  `json:"label"`
	Base    float64 `json:"base"`
	Bonus   float64 `json:"bonus"`
	Score   float64 `json:"score"`
	Reason  string  `json:"reason"`
}

type Query struct {
	Lat          float64
	Lon          float64
	At           time.Time
	ShoreBearing float64
	Tidal        bool
	WaterTempC   *float64
}

type WeatherProvider interface {
	Hourly(q Query, from, to time.Time) ([]Sample, error)
}

type TideProvider interface {
	Series(q Query, from, to time.Time) ([]TidePoint, error)
}

type ClassifiedError struct {
	Class   string
	Message string
}

func (e ClassifiedError) Error() string { return e.Class + ": " + e.Message }

const (
	ClassTransient = "TRANSIENT"
	ClassAuth      = "AUTH"
	ClassValidation = "VALIDATION"
	ClassBounds    = "BOUNDS"
)
