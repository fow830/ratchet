package main

import (
	"errors"
	"fmt"
	"testing"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{nil, exitOK},
		{violationErr(fmt.Errorf("drift")), exitViolation},
		{systemErr(fmt.Errorf("boom")), exitSystem},
		{fmt.Errorf("plain"), exitSystem},
		{fmt.Errorf("wrap: %w", violationErr(fmt.Errorf("x"))), exitViolation},
	}
	for _, tc := range tests {
		if got := exitCode(tc.err); got != tc.want {
			t.Fatalf("exitCode(%v)=%d want %d", tc.err, got, tc.want)
		}
	}
	var ce *codedError
	if !errors.As(violationErr(fmt.Errorf("x")), &ce) {
		t.Fatal("expected As")
	}
}
