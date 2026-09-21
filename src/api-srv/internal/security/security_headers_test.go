package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeadersMiddleware_SetsSecurityHeaders(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/example", nil)
	rec := httptest.NewRecorder()
	HeadersMiddleware(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("wrapped handler was not called")
	}

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for name, want := range headers {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("header %s = %q, want %q", name, got, want)
		}
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("Content-Security-Policy header not set")
	}
}

func TestHeadersMiddleware_SkipsFrameOptionsForAssets(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()
	HeadersMiddleware(next).ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Frame-Options"); got != "" {
		t.Errorf("X-Frame-Options for /assets/ path = %q, want empty", got)
	}
}

func TestHeadersMiddleware_NoCacheHeadersForHealthCheck(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	HeadersMiddleware(next).ServeHTTP(rec, req)

	if got := rec.Header().Get("Cache-Control"); got != "" {
		t.Errorf("Cache-Control for /health = %q, want empty", got)
	}
}

func TestHeadersMiddleware_NoCacheHeadersOnNonGET(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest(http.MethodPost, "/example", nil)
	rec := httptest.NewRecorder()
	HeadersMiddleware(next).ServeHTTP(rec, req)

	if got := rec.Header().Get("Cache-Control"); got != "" {
		t.Errorf("Cache-Control on POST = %q, want empty", got)
	}
}

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})
	mw := CORSMiddleware([]string{"https://allowed.example"})

	req := httptest.NewRequest(http.MethodGet, "/example", nil)
	req.Header.Set("Origin", "https://allowed.example")
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("wrapped handler was not called for allowed origin")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://allowed.example" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://allowed.example")
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", got, "true")
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want %q", got, "Origin")
	}
}

func TestCORSMiddleware_DisallowedOriginGetsNoHeaders(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})
	mw := CORSMiddleware([]string{"https://allowed.example"})

	req := httptest.NewRequest(http.MethodGet, "/example", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("wrapped handler was not called for disallowed origin")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin for disallowed origin = %q, want empty", got)
	}
}

func TestCORSMiddleware_PreflightFromAllowedOriginSkipsNext(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})
	mw := CORSMiddleware([]string{"https://allowed.example"})

	req := httptest.NewRequest(http.MethodOptions, "/example", nil)
	req.Header.Set("Origin", "https://allowed.example")
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Error("wrapped handler was called for an OPTIONS preflight, want it skipped")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("preflight status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://allowed.example" {
		t.Errorf("preflight Access-Control-Allow-Origin = %q, want %q", got, "https://allowed.example")
	}
}
