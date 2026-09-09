package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"context"
	"crypto/tls"
	"net"
	"net/http/httputil"
	"net/netip"
	"net/url"

	"github.com/caarlos0/env/v11"
	"tailscale.com/ipn"
)

func TestConfigDefaults(t *testing.T) {
	os.Clearenv()
	t.Setenv("HOSTNAME", "test-host")
	t.Setenv("TS_AUTHKEY", "test-key")

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
	t.Setenv("BACKEND", "http://192.168.1.100:3000")
	t.Setenv("FUNNEL", "true")
	t.Setenv("HOSTNAME", "custom-host")
	t.Setenv("HTTP_PORT", "8080")
	t.Setenv("HTTPS_PORT", "8443")
	t.Setenv("STATE_DIR", "/tmp/tsrp")
	t.Setenv("TS_AUTHKEY", "custom-key")
	t.Setenv("VERBOSE", "true")

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

// fakeConn is a net.Conn that only needs to satisfy the interface; the tests
// care solely about its dynamic type.
type fakeConn struct {
	net.Conn
}

func reqWithConn(t *testing.T, remoteAddr string, conn net.Conn) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	r.RemoteAddr = remoteAddr
	if conn != nil {
		r = r.WithContext(context.WithValue(r.Context(), ctxConn{}, conn))
	}
	return r
}

func TestClientIP(t *testing.T) {
	funnel := &ipn.FunnelConn{
		Conn: fakeConn{},
		Src:  netip.MustParseAddrPort("203.0.113.9:52000"),
	}

	tests := []struct {
		name       string
		remoteAddr string
		conn       net.Conn
		want       string
		wantOK     bool
	}{
		{
			name:       "funnel conn reports the originating client, not the relay",
			remoteAddr: "[fd7a:115c:a1e0::f701:f79c]:41234",
			conn:       funnel,
			want:       "203.0.113.9",
			wantOK:     true,
		},
		{
			name:       "funnel conn behind a TLS listener is unwrapped",
			remoteAddr: "[fd7a:115c:a1e0::f701:f79c]:41234",
			conn:       tls.Server(funnel, &tls.Config{}),
			want:       "203.0.113.9",
			wantOK:     true,
		},
		{
			name:       "direct tailnet peer uses its own address",
			remoteAddr: "100.125.213.97:41234",
			conn:       fakeConn{},
			want:       "100.125.213.97",
			wantOK:     true,
		},
		{
			name:       "no conn in context falls back to RemoteAddr",
			remoteAddr: "100.125.213.97:41234",
			want:       "100.125.213.97",
			wantOK:     true,
		},
		{
			name:       "unparseable RemoteAddr yields nothing",
			remoteAddr: "garbage",
			want:       "",
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := clientIP(reqWithConn(t, tt.remoteAddr, tt.conn))
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("clientIP() = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// The backend must receive exactly the address we vouch for, never a value the
// client supplied.
func TestProxySetsTrustedForwardedFor(t *testing.T) {
	var gotXFF []string
	var gotHost string
	backend := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotXFF = r.Header["X-Forwarded-For"]
		gotHost = r.Host
	}))
	defer backend.Close()

	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(backendURL)
			pr.Out.Host = pr.In.Host
			if ip, ok := clientIP(pr.In); ok {
				pr.Out.Header.Set("X-Forwarded-For", ip)
			} else {
				pr.Out.Header.Del("X-Forwarded-For")
			}
		},
	}

	funnel := &ipn.FunnelConn{
		Conn: fakeConn{},
		Src:  netip.MustParseAddrPort("203.0.113.9:52000"),
	}

	req := reqWithConn(t, "[fd7a:115c:a1e0::f701:f79c]:41234", funnel)
	req.Host = "svc.example.ts.net"
	// A client trying to pass itself off as someone else.
	req.Header.Set("X-Forwarded-For", "198.51.100.77")

	rp.ServeHTTP(httptest.NewRecorder(), req)

	if len(gotXFF) != 1 || gotXFF[0] != "203.0.113.9" {
		t.Errorf("X-Forwarded-For = %v, want [203.0.113.9]", gotXFF)
	}
	if gotHost != "svc.example.ts.net" {
		t.Errorf("Host = %q, want the client's original host", gotHost)
	}
}
