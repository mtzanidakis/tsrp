package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/caarlos0/env/v11"
)

func TestConfigDefaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("HOSTNAME", "test-host")
	os.Setenv("TS_AUTHKEY", "test-key")

	cfg := config{}
	opts := env.Options{RequiredIfNoDef: true}

	if err := env.ParseWithOptions(&cfg, opts); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if cfg.Backend != "http://127.0.0.1:8080" {
		t.Errorf("Expected Backend to be 'http://127.0.0.1:8080', got '%s'", cfg.Backend)
	}
	if cfg.Funnel != false {
		t.Errorf("Expected Funnel to be false, got %v", cfg.Funnel)
	}
	if cfg.HTTPPort != 80 {
		t.Errorf("Expected HTTPPort to be 80, got %d", cfg.HTTPPort)
	}
	if cfg.HTTPSPort != 443 {
		t.Errorf("Expected HTTPSPort to be 443, got %d", cfg.HTTPSPort)
	}
	if cfg.StateDir != "/var/lib/tsrp" {
		t.Errorf("Expected StateDir to be '/var/lib/tsrp', got '%s'", cfg.StateDir)
	}
	if cfg.Verbose != false {
		t.Errorf("Expected Verbose to be false, got %v", cfg.Verbose)
	}
}

func TestConfigCustomValues(t *testing.T) {
	os.Clearenv()
	os.Setenv("BACKEND", "http://192.168.1.100:3000")
	os.Setenv("FUNNEL", "true")
	os.Setenv("HOSTNAME", "custom-host")
	os.Setenv("HTTP_PORT", "8080")
	os.Setenv("HTTPS_PORT", "8443")
	os.Setenv("STATE_DIR", "/tmp/tsrp")
	os.Setenv("TS_AUTHKEY", "custom-key")
	os.Setenv("VERBOSE", "true")

	cfg := config{}
	opts := env.Options{RequiredIfNoDef: true}

	if err := env.ParseWithOptions(&cfg, opts); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if cfg.Backend != "http://192.168.1.100:3000" {
		t.Errorf("Expected Backend to be 'http://192.168.1.100:3000', got '%s'", cfg.Backend)
	}
	if cfg.Funnel != true {
		t.Errorf("Expected Funnel to be true, got %v", cfg.Funnel)
	}
	if cfg.Hostname != "custom-host" {
		t.Errorf("Expected Hostname to be 'custom-host', got '%s'", cfg.Hostname)
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("Expected HTTPPort to be 8080, got %d", cfg.HTTPPort)
	}
	if cfg.HTTPSPort != 8443 {
		t.Errorf("Expected HTTPSPort to be 8443, got %d", cfg.HTTPSPort)
	}
	if cfg.StateDir != "/tmp/tsrp" {
		t.Errorf("Expected StateDir to be '/tmp/tsrp', got '%s'", cfg.StateDir)
	}
	if cfg.TSAuthkey != "custom-key" {
		t.Errorf("Expected TSAuthkey to be 'custom-key', got '%s'", cfg.TSAuthkey)
	}
	if cfg.Verbose != true {
		t.Errorf("Expected Verbose to be true, got %v", cfg.Verbose)
	}
}

func TestRedirectHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpsURL := "https://" + r.Host + r.RequestURI
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})

	tests := []struct {
		name        string
		host        string
		requestURI  string
		expectedURL string
	}{
		{
			name:        "basic redirect",
			host:        "example.com",
			requestURI:  "/test",
			expectedURL: "https://example.com/test",
		},
		{
			name:        "redirect with query params",
			host:        "example.com",
			requestURI:  "/test?param=value",
			expectedURL: "https://example.com/test?param=value",
		},
		{
			name:        "redirect root path",
			host:        "example.com",
			requestURI:  "/",
			expectedURL: "https://example.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://"+tt.host+tt.requestURI, nil)
			req.Host = tt.host
			req.RequestURI = tt.requestURI
			
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusMovedPermanently {
				t.Errorf("Expected status %d, got %d", http.StatusMovedPermanently, rr.Code)
			}

			location := rr.Header().Get("Location")
			if location != tt.expectedURL {
				t.Errorf("Expected Location header to be '%s', got '%s'", tt.expectedURL, location)
			}
		})
	}
}

func TestReverseProxyErrorHandler(t *testing.T) {
	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "502 bad gateway", http.StatusBadGateway)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	
	errorHandler(rr, req, http.ErrServerClosed)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("Expected status %d, got %d", http.StatusBadGateway, rr.Code)
	}

	expectedBody := "502 bad gateway\n"
	if rr.Body.String() != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, rr.Body.String())
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/plain; charset=utf-8', got '%s'", contentType)
	}
}