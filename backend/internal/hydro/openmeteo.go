package hydro

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type OpenMeteo struct {
	Client  *http.Client
	BaseURL string
}

type omResponse struct {
	Error   bool   `json:"error"`
	Reason  string `json:"reason"`
	Hourly  *omHourly `json:"hourly"`
}

type omHourly struct {
	Time             []string   `json:"time"`
	PressureMSL      []*float64 `json:"pressure_msl"`
	Temperature2M    []*float64 `json:"temperature_2m"`
	WindSpeed10M     []*float64 `json:"windspeed_10m"`
	WindDirection10M []*float64 `json:"winddirection_10m"`
}

func (p OpenMeteo) Hourly(q Query, from, to time.Time) ([]Sample, error) {
	if q.Lat < -90 || q.Lat > 90 || q.Lon < -180 || q.Lon > 180 {
		return nil, ClassifiedError{Class: ClassBounds, Message: "coordinate out of range"}
	}
	base := p.BaseURL
	if base == "" {
		base = "https://archive-api.open-meteo.com/v1/archive"
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	}
	from = from.UTC()
	to = to.UTC()
	u, _ := url.Parse(base)
	qs := u.Query()
	qs.Set("latitude", fmt.Sprintf("%.4f", q.Lat))
	qs.Set("longitude", fmt.Sprintf("%.4f", q.Lon))
	qs.Set("start_date", from.Format("2006-01-02"))
	qs.Set("end_date", to.Format("2006-01-02"))
	qs.Set("hourly", "pressure_msl,temperature_2m,windspeed_10m,winddirection_10m")
	qs.Set("timezone", "UTC")
	u.RawQuery = qs.Encode()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, ClassifiedError{Class: ClassValidation, Message: err.Error()}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, ClassifiedError{Class: ClassTransient, Message: err.Error()}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, ClassifiedError{Class: ClassTransient, Message: err.Error()}
	}
	if resp.StatusCode >= 500 {
		return nil, ClassifiedError{Class: ClassTransient, Message: fmt.Sprintf("open-meteo %d", resp.StatusCode)}
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, ClassifiedError{Class: ClassAuth, Message: string(body)}
	}
	if resp.StatusCode >= 400 {
		return nil, ClassifiedError{Class: ClassValidation, Message: fmt.Sprintf("open-meteo %d: %s", resp.StatusCode, body)}
	}
	var parsed omResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, ClassifiedError{Class: ClassValidation, Message: "open-meteo json: " + err.Error()}
	}
	if parsed.Error {
		return nil, ClassifiedError{Class: ClassValidation, Message: parsed.Reason}
	}
	if parsed.Hourly == nil {
		return nil, ClassifiedError{Class: ClassValidation, Message: "missing hourly"}
	}
	h := parsed.Hourly
	n := len(h.Time)
	if n == 0 || len(h.PressureMSL) != n || len(h.Temperature2M) != n || len(h.WindSpeed10M) != n || len(h.WindDirection10M) != n {
		return nil, ClassifiedError{Class: ClassValidation, Message: "hourly series length mismatch"}
	}
	out := make([]Sample, 0, n)
	for i := 0; i < n; i++ {
		if h.PressureMSL[i] == nil || h.Temperature2M[i] == nil || h.WindSpeed10M[i] == nil || h.WindDirection10M[i] == nil {
			continue
		}
		at, err := time.ParseInLocation("2006-01-02T15:04", h.Time[i], time.UTC)
		if err != nil {
			at, err = time.Parse(time.RFC3339, h.Time[i])
			if err != nil {
				return nil, ClassifiedError{Class: ClassValidation, Message: "bad hourly time " + h.Time[i]}
			}
		}
		out = append(out, Sample{
			At:          at.UTC(),
			PressureHPa: *h.PressureMSL[i],
			TempC:       *h.Temperature2M[i],
			WindDirDeg:  *h.WindDirection10M[i],
			WindMS:      *h.WindSpeed10M[i] / 3.6,
		})
	}
	if len(out) == 0 {
		return nil, ClassifiedError{Class: ClassValidation, Message: "empty usable hourly series"}
	}
	return out, nil
}
