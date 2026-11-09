package main

import (
	"context"
	"net/http"
	"time"
)

type State string

const (
	Up      State = "up"      // the server answered (a login redirect counts)
	Warming State = "warming" // something answered but the game side is not ready (5xx from a proxy or container)
	Down    State = "down"    // no connection
)

// Probe asks GET <origin>/watch once, without following redirects (an unauthenticated visit may redirect to the
// login page; that is "up"). It never sends cookies or credentials.
func Probe(ctx context.Context, client *http.Client, origin string) State {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+"/watch", nil)
	if err != nil {
		return Down
	}
	resp, err := client.Do(req)
	if err != nil {
		return Down
	}
	resp.Body.Close()
	if resp.StatusCode >= 500 {
		return Warming
	}
	return Up
}

// NewProbeClient does not follow redirects.
func NewProbeClient() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
