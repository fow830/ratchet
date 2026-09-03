package smoke_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fow830/ratchet/pkg/smoke"
)

func TestLiveGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	res, err := smoke.GET(srv.URL, smoke.Options{Timeout: 2 * time.Second, WantStatus: http.StatusOK})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("%+v", res)
	}
}

func TestLiveGETWrongStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()
	_, err := smoke.GET(srv.URL, smoke.Options{WantStatus: http.StatusOK})
	if err == nil {
		t.Fatal("expected error")
	}
}
