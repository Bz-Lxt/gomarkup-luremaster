package hydro_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"luremaster/internal/hydro"
)

func TestMalformedWeatherPayloadDoesNotConsumeRetryBudget(t *testing.T) {
	var requests atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hourly":`))
	}))
	defer upstream.Close()

	provider := hydro.OpenMeteo{Client: upstream.Client(), BaseURL: upstream.URL}
	engine := hydro.NewEngine(provider, hydro.HarmonicTide{})
	query := hydro.Query{
		Lat: 30.25,
		Lon: 122.18,
		At:  time.Date(2026, 8, 25, 5, 30, 0, 0, time.UTC),
	}

	attempts := 0
	class := ""
	var lastErr error
	for {
		_, lastErr = engine.Retro(query)
		if lastErr == nil {
			t.Fatal("malformed upstream payload unexpectedly succeeded")
		}
		attempts++
		class = hydro.ClassifyError(lastErr)
		if !hydro.ShouldRetry(class, attempts) {
			break
		}
	}

	var classified hydro.ClassifiedError
	if !errors.As(lastErr, &classified) || classified.Class != hydro.ClassValidation {
		t.Fatalf("error chain lost validation cause: %v", lastErr)
	}
	if class != hydro.ClassValidation {
		t.Fatalf("malformed payload class = %s, want %s", class, hydro.ClassValidation)
	}
	if attempts != 1 || requests.Load() != 1 {
		t.Fatalf("malformed payload attempts = %d, requests = %d; want 1, 1", attempts, requests.Load())
	}
}
