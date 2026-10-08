package main

import (
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Shutdown uses the same exact request contract as the other mutations.
// A valid local token alone must not make malformed requests authoritative.
func TestReviewShutdownRejectsMalformedAuthenticatedRequests(t *testing.T) {
	a := NewApp()
	token := sessionFor(t, a)
	var shutdownCalls atomic.Int32
	handler := newWebHandler(a, 8088, func() { shutdownCalls.Add(1) })
	for _, tc := range []struct {
		name, body, contentType, authorization, origin string
	}{
		{"mime prefix", `{}`, "application/json-bogus", "Bearer " + token, ""},
		{"plain token", `{}`, "application/json", token, ""},
		{"origin path", `{}`, "application/json", "Bearer " + token, "http://127.0.0.1:8088/path"},
		{"origin userinfo", `{}`, "application/json", "Bearer " + token, "http://user@127.0.0.1:8088"},
		{"origin query", `{}`, "application/json", "Bearer " + token, "http://127.0.0.1:8088?x=1"},
		{"null body", `null`, "application/json", "Bearer " + token, ""},
		{"multiple bodies", `{} {}`, "application/json", "Bearer " + token, ""},
		{"unknown field", `{"unexpected":1}`, "application/json", "Bearer " + token, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "http://127.0.0.1:8088/api/shutdown", strings.NewReader(tc.body))
			request.RemoteAddr = "127.0.0.1:1234"
			request.Header.Set("Content-Type", tc.contentType)
			request.Header.Set("X-Nexo-Client", "1")
			request.Header.Set("Authorization", tc.authorization)
			if tc.origin != "" {
				request.Header.Set("Origin", tc.origin)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code < 400 || !strings.Contains(response.Body.String(), `"error"`) {
				t.Fatalf("malformed shutdown accepted: status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
	if calls := shutdownCalls.Load(); calls != 0 {
		t.Fatalf("malformed requests scheduled shutdown %d times", calls)
	}
}
