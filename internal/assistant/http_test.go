package assistant

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestTLSFallbackReplaysBodyAndSticks(t *testing.T) {
	primaryCalls := 0
	fallbackCalls := 0
	fallbackBody := ""

	primary := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		primaryCalls++
		if _, err := io.ReadAll(req.Body); err != nil {
			t.Fatal(err)
		}
		return nil, errors.New("Post \"https://example.com\": tls: server did not echo the legacy session ID")
	})
	fallback := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		fallbackCalls++
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		fallbackBody = string(raw)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	tr := &tlsFallbackTransport{primary: primary, fallback: fallback}
	req, err := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if primaryCalls != 1 || fallbackCalls != 1 || fallbackBody != "payload" {
		t.Fatalf("primary=%d fallback=%d body=%q", primaryCalls, fallbackCalls, fallbackBody)
	}

	req2, err := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader("second"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err = tr.RoundTrip(req2)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if primaryCalls != 1 || fallbackCalls != 2 {
		t.Fatalf("sticky fallback failed: primary=%d fallback=%d", primaryCalls, fallbackCalls)
	}
}

func TestTLSFallbackDoesNotCatchOtherErrors(t *testing.T) {
	fallbackCalls := 0
	tr := &tlsFallbackTransport{
		primary: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("connection reset by peer")
		}),
		fallback: roundTripFunc(func(*http.Request) (*http.Response, error) {
			fallbackCalls++
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
		}),
	}
	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tr.RoundTrip(req); err == nil {
		t.Fatal("expected error")
	}
	if fallbackCalls != 0 {
		t.Fatalf("fallback called %d times", fallbackCalls)
	}
}

func TestLegacySessionIDErrorIsNotTransient(t *testing.T) {
	err := errors.New("tls: server did not echo the legacy session ID")
	if !IsLegacySessionIDError(err) {
		t.Fatal("legacy session error not recognized")
	}
	if IsTransientError(err) {
		t.Fatal("legacy session error must not be retried")
	}
}
