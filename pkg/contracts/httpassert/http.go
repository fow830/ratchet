// Package httpassert provides HTTP contract test helpers.
package httpassert

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// AssertStatus fails the test when status codes differ.
func AssertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if err := CheckStatus(resp, want); err != nil {
		t.Fatal(err)
	}
}

// CheckStatus returns an error when status codes differ.
func CheckStatus(resp *http.Response, want int) error {
	if resp == nil {
		return fmt.Errorf("nil response")
	}
	if resp.StatusCode != want {
		return fmt.Errorf("status: got %d want %d", resp.StatusCode, want)
	}
	return nil
}

// AssertHeader fails when header value differs.
func AssertHeader(t *testing.T, resp *http.Response, key, want string) {
	t.Helper()
	if err := CheckHeader(resp, key, want); err != nil {
		t.Fatal(err)
	}
}

// CheckHeader returns an error when header value differs.
func CheckHeader(resp *http.Response, key, want string) error {
	if resp == nil {
		return fmt.Errorf("nil response")
	}
	got := resp.Header.Get(key)
	if got != want {
		return fmt.Errorf("header %q: got %q want %q", key, got, want)
	}
	return nil
}

// NewTestServer returns an httptest.Server that closes on test cleanup.
func NewTestServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

// AssertMethod fails when HTTP method differs.
func AssertMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if r.Method != want {
		t.Fatalf("method: got %q want %q", r.Method, want)
	}
}
