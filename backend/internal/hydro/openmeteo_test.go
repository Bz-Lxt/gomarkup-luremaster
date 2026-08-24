package hydro

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenMeteoParsesVerifiedContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("hourly") == "" || q.Get("timezone") != "UTC" {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":true,"reason":"missing"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "latitude":30.26,"longitude":122.14,"utc_offset_seconds":0,
		  "hourly":{
		    "time":["2026-08-20T00:00","2026-08-20T01:00"],
		    "pressure_msl":[1009.2,1008.8],
		    "temperature_2m":[29.0,29.3],
		    "windspeed_10m":[18.0,21.6],
		    "winddirection_10m":[98,93]
		  }
		}`))
	}))
	defer srv.Close()
	p := OpenMeteo{Client: srv.Client(), BaseURL: srv.URL}
	got, err := p.Hourly(Query{Lat: 30.25, Lon: 122.18}, time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 20, 2, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].WindMS < 4.9 || got[0].WindMS > 5.1 {
		t.Fatalf("km/h to m/s %v", got[0].WindMS)
	}
}

func TestOpenMeteoValidationNotRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":true,"reason":"bad lat"}`))
	}))
	defer srv.Close()
	p := OpenMeteo{Client: srv.Client(), BaseURL: srv.URL}
	_, err := p.Hourly(Query{Lat: 30, Lon: 120}, time.Now(), time.Now().Add(time.Hour))
	if ClassifyError(err) != ClassValidation {
		t.Fatalf("class %s err=%v", ClassifyError(err), err)
	}
	if Retryable(ClassifyError(err)) {
		t.Fatal("must not retry validation")
	}
}

func TestOpenMeteoAuthNotRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte("no"))
	}))
	defer srv.Close()
	p := OpenMeteo{Client: srv.Client(), BaseURL: srv.URL}
	_, err := p.Hourly(Query{Lat: 1, Lon: 1}, time.Now(), time.Now().Add(time.Hour))
	if ClassifyError(err) != ClassAuth || Retryable(ClassAuth) {
		t.Fatalf("auth class %s", ClassifyError(err))
	}
}

func TestOpenMeteoBounds(t *testing.T) {
	_, err := OpenMeteo{}.Hourly(Query{Lat: 200, Lon: 0}, time.Now(), time.Now())
	if ClassifyError(err) != ClassBounds {
		t.Fatalf("%v", err)
	}
}

func TestOpenMeteoTransient5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()
	p := OpenMeteo{Client: srv.Client(), BaseURL: srv.URL}
	_, err := p.Hourly(Query{Lat: 1, Lon: 1}, time.Now(), time.Now().Add(time.Hour))
	if !Retryable(ClassifyError(err)) {
		t.Fatalf("5xx should be transient: %v", err)
	}
}
