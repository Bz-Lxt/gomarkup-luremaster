package hydro

import (
	"testing"
	"time"
)

type countingWeather struct {
	n     int
	class string
	msg   string
}

func (c *countingWeather) Hourly(_ Query, _, _ time.Time) ([]Sample, error) {
	c.n++
	return nil, ClassifiedError{Class: c.class, Message: c.msg}
}

func runUntilStop(w WeatherProvider) (calls, retries int) {
	eng := NewEngine(w, HarmonicTide{})
	q := Query{Lat: 30.25, Lon: 122.18, At: time.Date(2026, 8, 20, 6, 15, 0, 0, time.UTC)}
	attempts := 0
	for {
		_, err := eng.Retro(q)
		attempts++
		class := ClassifyError(err)
		if !ShouldRetry(class, attempts) {
			break
		}
	}
	if cw, ok := w.(*countingWeather); ok {
		calls = cw.n
	}
	retries = attempts - 1
	if retries < 0 {
		retries = 0
	}
	return calls, retries
}

func TestAuthAndValidationRetryCountZero(t *testing.T) {
	cases := []struct {
		class string
		msg   string
	}{
		{ClassAuth, "denied"},
		{ClassValidation, "bad request"},
	}
	for _, tc := range cases {
		p := &countingWeather{class: tc.class, msg: tc.msg}
		calls, retries := runUntilStop(p)
		if retries != 0 {
			t.Fatalf("%s retries=%d want 0", tc.class, retries)
		}
		if calls != 1 {
			t.Fatalf("%s provider calls=%d want 1", tc.class, calls)
		}
	}
}

func TestTransientRetriesUntilMaxAttempts(t *testing.T) {
	p := &countingWeather{class: ClassTransient, msg: "timeout"}
	calls, retries := runUntilStop(p)
	if calls != MaxAttempts {
		t.Fatalf("transient calls=%d want %d", calls, MaxAttempts)
	}
	if retries != MaxAttempts-1 {
		t.Fatalf("transient retries=%d want %d", retries, MaxAttempts-1)
	}
}
