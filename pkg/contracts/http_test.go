package contracts_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fow830/ratchet/pkg/contracts/httpassert"
)

func TestAssertStatusOK(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.WriteHeader(http.StatusOK)
	httpassert.AssertStatus(t, rec.Result(), http.StatusOK)
}

func TestCheckStatusFails(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.WriteHeader(http.StatusTeapot)
	if err := httpassert.CheckStatus(rec.Result(), http.StatusOK); err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	if err := httpassert.CheckHeader(rec.Result(), "Content-Type", "application/json"); err != nil {
		t.Fatal(err)
	}
}

func TestNewTestServer(t *testing.T) {
	srv := httpassert.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if err := httpassert.CheckStatus(resp, http.StatusNoContent); err != nil {
		t.Fatal(err)
	}
}
