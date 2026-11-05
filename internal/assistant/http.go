package assistant

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
)

type tlsFallbackTransport struct {
	primary        http.RoundTripper
	fallback       http.RoundTripper
	fallbackNeeded atomic.Bool
}

func (t *tlsFallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.fallbackNeeded.Load() {
		return t.fallback.RoundTrip(req)
	}

	resp, err := t.primary.RoundTrip(req)
	if err == nil || !IsLegacySessionIDError(err) {
		return resp, err
	}
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	retry, replayErr := replayRequest(req)
	if replayErr != nil {
		return nil, err
	}
	resp, err = t.fallback.RoundTrip(retry)
	if err != nil {
		return nil, fmt.Errorf("TLS 1.2 fallback failed: %w", err)
	}
	t.fallbackNeeded.Store(true)
	return resp, nil
}

func replayRequest(req *http.Request) (*http.Request, error) {
	retry := req.Clone(req.Context())
	if req.Body == nil {
		return retry, nil
	}
	if req.GetBody == nil {
		return nil, errors.New("request body cannot be replayed")
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	retry.Body = body
	return retry, nil
}

func newAssistantHTTPClient() *http.Client {
	primary := http.DefaultTransport.(*http.Transport).Clone()
	primary.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.X25519, tls.CurveP256, tls.CurveP384,
		},
	}
	fallback := http.DefaultTransport.(*http.Transport).Clone()
	fallback.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.X25519, tls.CurveP256, tls.CurveP384,
		},
	}
	return &http.Client{Transport: &tlsFallbackTransport{primary: primary, fallback: fallback}}
}

var assistantHTTPClient = newAssistantHTTPClient()

// IsLegacySessionIDError reports the TLS 1.3 middlebox compatibility violation.
func IsLegacySessionIDError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "did not echo the legacy session id")
}
