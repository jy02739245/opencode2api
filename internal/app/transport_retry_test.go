package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestTransportErrorRetryable(t *testing.T) {
	cases := []struct {
		name    string
		body    int64
		attempt int
		want    bool
	}{
		{"small first attempt", 1024, 0, true},
		{"small last attempt", 1024, maxUpstreamRetries - 1, false},
		{"at limit first attempt", largeBodyTransportRetryLimit, 0, true},
		{"over limit first attempt", largeBodyTransportRetryLimit + 1, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := transportErrorRetryable(tc.body, tc.attempt); got != tc.want {
				t.Fatalf("transportErrorRetryable(%d, %d) = %v, want %v", tc.body, tc.attempt, got, tc.want)
			}
		})
	}
}

func TestCallOpenCodeEndpoint_LargeBodyTransportErrorDoesNotRetry(t *testing.T) {
	transport := installFakeOpenCodeClient(t, []fakeUpstreamResponse{
		{err: errors.New("transport boom")},
		{err: errors.New("transport boom again")},
	})
	payload := `{"model":"test-model","messages":[{"role":"user","content":"` +
		strings.Repeat("x", largeBodyTransportRetryLimit) + `"}]}`

	_, status, _, err := callOpenCodeEndpoint(context.Background(), "chat/completions",
		[]byte(payload), "test-model", UpstreamAuth{Mode: AuthRouteAuto, Token: "sk-x"})
	if err == nil {
		t.Fatalf("expected transport error, got status=%d", status)
	}
	if len(transport.requestedModels) != 1 {
		t.Fatalf("large body must not be retried, got %d attempts", len(transport.requestedModels))
	}
}
