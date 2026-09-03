package httpassert_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fow830/ratchet/pkg/contracts/httpassert"
)

func TestAssertStatusAndHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Ratchet", "1")
	rec.WriteHeader(http.StatusCreated)
	resp := rec.Result()
	httpassert.AssertStatus(t, resp, http.StatusCreated)
	httpassert.AssertHeader(t, resp, "X-Ratchet", "1")
}

func TestAssertMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	httpassert.AssertMethod(t, req, http.MethodPost)
}
