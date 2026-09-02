package main

import "errors"

// Exit codes per Phase 3 TZ.
const (
	exitOK        = 0
	exitViolation = 1
	exitSystem    = 2
)

type codedError struct {
	code int
	err  error
}

func (e *codedError) Error() string { return e.err.Error() }
func (e *codedError) Unwrap() error { return e.err }

func violationErr(err error) error { return &codedError{code: exitViolation, err: err} }
func systemErr(err error) error    { return &codedError{code: exitSystem, err: err} }

func exitCode(err error) int {
	if err == nil {
		return exitOK
	}
	var ce *codedError
	if errors.As(err, &ce) {
		return ce.code
	}
	return exitSystem
}
