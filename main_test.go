package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebAssetsEmbeddedAndUncached(t *testing.T) {
	h := newWebHandler(NewApp(), 8088, nil)
	for _, path := range []string{"/", "/style.css", "/app.js", "/pieces.js"} {
		req := httptest.NewRequest("GET", "http://127.0.0.1:8088"+path, nil)
		req.RemoteAddr = "127.0.0.1:1234"
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if res.Code != 200 || res.Body.Len() == 0 {
			t.Fatalf("asset %s: %d %s", path, res.Code, res.Body.String())
		}
		if res.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("browser should not persist app resources")
		}
	}
}

func TestShutdownRequiresLocalAuthenticatedClient(t *testing.T) {
	app := NewApp()
	closed := make(chan bool, 1)
	h := newWebHandler(app, 8088, func() { closed <- true })
	res := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "http://127.0.0.1:8088/api/session", strings.NewReader("{}"))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Nexo-Client", "1")
	app.ServeHTTP(res, req)
	var session struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &session); err != nil || session.Token == "" {
		t.Fatal("session creation failed")
	}
	// No authentication must never shut down the host.
	for _, addr := range []string{"127.0.0.1:1234", "192.168.1.20:2345"} {
		req = httptest.NewRequest("POST", "http://127.0.0.1:8088/api/shutdown", strings.NewReader("{}"))
		req.RemoteAddr = addr
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Nexo-Client", "1")
		res = httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if res.Code == 200 {
			t.Fatal("unauthenticated shutdown accepted")
		}
	}
	req = httptest.NewRequest("POST", "http://127.0.0.1:8088/api/shutdown", strings.NewReader("{}"))
	req.RemoteAddr = "192.168.1.20:2345"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Nexo-Client", "1")
	req.Header.Set("Authorization", "Bearer "+session.Token)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code == 200 {
		t.Fatal("LAN guest shutdown accepted")
	}
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Origin", "http://evil.example")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code == 200 {
		t.Fatal("cross site shutdown accepted")
	}
	req.Header.Del("Origin")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != 200 {
		t.Fatalf("host cannot shutdown: %d %s", res.Code, res.Body.String())
	}
	<-closed
}

func TestPublicRequestsRejected(t *testing.T) {
	h := newWebHandler(NewApp(), 8088, nil)
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8088/", nil)
	req.RemoteAddr = "8.8.8.8:1234"
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatal("public address accepted")
	}
}

func TestHostRejectsUnrelatedDomain(t *testing.T) {
	h := newWebHandler(NewApp(), 8088, nil)
	req := httptest.NewRequest(http.MethodGet, "http://unrelated.example:8088/api/info", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatal("unrelated hostname accepted")
	}
}
