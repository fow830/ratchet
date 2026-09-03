// Package smoke provides production/staging HTTP smoke contract helpers.
package smoke

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Options configures a smoke probe.
type Options struct {
	Timeout    time.Duration
	WantStatus int
	WantBody   string
	Header     http.Header
}

// Result is the outcome of a smoke probe.
type Result struct {
	OK         bool
	StatusCode int
	Body       string
	Latency    time.Duration
}

// GET probes url and validates status/body.
func GET(url string, opts Options) (Result, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.WantStatus == 0 {
		opts.WantStatus = http.StatusOK
	}
	client := &http.Client{Timeout: opts.Timeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Result{}, err
	}
	for k, vals := range opts.Header {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	res := Result{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Latency:    time.Since(start),
		OK:         resp.StatusCode == opts.WantStatus,
	}
	if opts.WantBody != "" && res.Body != opts.WantBody {
		res.OK = false
		return res, fmt.Errorf("body: got %q want %q", res.Body, opts.WantBody)
	}
	if !res.OK {
		return res, fmt.Errorf("status: got %d want %d", res.StatusCode, opts.WantStatus)
	}
	return res, nil
}
