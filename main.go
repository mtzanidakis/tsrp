package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/caarlos0/env/v11"
	"tailscale.com/tsnet"
)

type config struct {
	Backend   string `env:"BACKEND" envDefault:"http://127.0.0.1:8080"`
	Hostname  string `env:"HOSTNAME"`
	HTTPPort  int    `env:"HTTP_PORT" envDefault:"80"`
	HTTPSPort int    `env:"HTTPS_PORT" envDefault:"443"`
	StateDir  string `env:"STATE_DIR,expand" envDefault:"/var/lib/tsrp"`
	TSAuthkey string `env:"TS_AUTHKEY"`
	Verbose   bool   `env:"VERBOSE" envDefault:"false"`
}

func main() {
	cfg := config{}
	opts := env.Options{RequiredIfNoDef: true}

	if err := env.ParseWithOptions(&cfg, opts); err != nil {
		log.Fatal(err)
	}

	backendUrl, err := url.Parse(cfg.Backend)
	if err != nil {
		log.Fatal(err)
	}

	tss := &tsnet.Server{
		AuthKey:  cfg.TSAuthkey,
		Dir:      fmt.Sprintf("%s/%s", cfg.StateDir, cfg.Hostname),
		Hostname: cfg.Hostname,
	}
	if !cfg.Verbose {
		tss.Logf = func(string, ...any) {}
	}
	defer tss.Close()

	// Start HTTP listener for redirects
	httpLn, err := tss.Listen("tcp", fmt.Sprintf(":%d", cfg.HTTPPort))
	if err != nil {
		log.Fatal(err)
	}
	defer httpLn.Close()

	// Start HTTPS listener
	tlsLn, err := tss.ListenTLS("tcp", fmt.Sprintf(":%d", cfg.HTTPSPort))
	if err != nil {
		log.Fatal(err)
	}
	defer tlsLn.Close()

	// Set up reverse proxy
	rp := httputil.NewSingleHostReverseProxy(backendUrl)
	rp.ErrorHandler = func(
		w http.ResponseWriter,
		r *http.Request,
		err error,
	) {
		http.Error(w, "502 bad gateway", http.StatusBadGateway)
		log.Printf("backend error: %s", err)
	}

	// HTTP redirect handler
	redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpsURL := fmt.Sprintf("https://%s%s", r.Host, r.RequestURI)
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})

	// Start both servers
	log.Printf("starting HTTP redirect server on port %d", cfg.HTTPPort)
	go func() {
		if err := http.Serve(httpLn, redirectHandler); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Printf("starting HTTPS reverse proxy to %s on port %d", cfg.Backend, cfg.HTTPSPort)
	log.Fatal(http.Serve(tlsLn, rp))
}
