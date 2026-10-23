package app

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxRequestBodyBytes_Config(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want int64
	}{
		{"default", "", 64 * 1024 * 1024},
		{"override", "8", 8 * 1024 * 1024},
		{"invalid falls back", "bogus", 64 * 1024 * 1024},
		{"zero falls back", "0", 64 * 1024 * 1024},
		{"negative falls back", "-1", 64 * 1024 * 1024},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OPENCODE2API_MAX_REQUEST_BODY_MB", tc.env)
			if got := maxRequestBodyBytes(); got != tc.want {
				t.Fatalf("maxRequestBodyBytes() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestReadJSONRequestBody_OversizedContentLengthReturns413(t *testing.T) {
	t.Setenv("OPENCODE2API_MAX_REQUEST_BODY_MB", "1")
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		bytes.NewReader(make([]byte, 1024*1024+1)))
	w := httptest.NewRecorder()

	_, _, ok := readJSONRequestBody(w, req)
	if ok {
		t.Fatal("oversized body must not be accepted")
	}
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Request body too large") {
		t.Fatalf("body = %q, want explicit size error", w.Body.String())
	}
}

func TestReadRequestBody_ChunkedOversizeReturns413(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		io.NopCloser(bytes.NewReader(make([]byte, 1025))))
	req.ContentLength = -1 // 模拟 chunked：没有 Content-Length 预检
	w := httptest.NewRecorder()

	_, ok := readRequestBody(w, req, 1024)
	if ok {
		t.Fatal("chunked oversized body must not be accepted")
	}
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", w.Code)
	}
}

func TestReadRequestBody_UnderLimitPreservesBody(t *testing.T) {
	want := []byte(`{"model":"m"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(want))
	w := httptest.NewRecorder()

	got, ok := readRequestBody(w, req, 1024)
	if !ok {
		t.Fatalf("readRequestBody failed: %s", w.Body.String())
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
