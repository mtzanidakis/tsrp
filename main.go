package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
	"tailscale.com/tsnet"
)

type config struct {
	Backend   string `env:"BACKEND" envDefault:"http://127.0.0.1:8080"`
	Funnel    bool   `env:"FUNNEL" envDefault:"false"`
	Hostname  string `env:"HOSTNAME"`
	HTTPPort  int    `env:"HTTP_PORT" envDefault:"80"`
	HTTPSPort int    `env:"HTTPS_PORT" envDefault:"443"`
	StateDir  string `env:"STATE_DIR,expand" envDefault:"/var/lib/tsrp"`
	TSAuthkey string `env:"TS_AUTHKEY"`
	Verbose   bool   `env:"VERBOSE" envDefault:"false"`
}

type bufferPool struct {
	pool sync.Pool
}

func (bp *bufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

func (bp *bufferPool) Put(buf []byte) {
	bp.pool.Put(buf)
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
	var (
		tlsLn   net.Listener
		logMess string
	)

	if cfg.Funnel {
		// if cfg.Port is not 443 or 8443, or 10000 fail
		if cfg.HTTPSPort != 443 && cfg.HTTPSPort != 8443 && cfg.HTTPSPort != 10000 {
			log.Fatal("Funnel mode requires port 443, 8443 or 10000")
		}

		tlsLn, err = tss.ListenFunnel("tcp", fmt.Sprintf(":%d", cfg.HTTPSPort))
		if err != nil {
			log.Fatal(err)
		}
		logMess = " [funnel]"
	} else {
		tlsLn, err = tss.ListenTLS("tcp", fmt.Sprintf(":%d", cfg.HTTPSPort))
		if err != nil {
			log.Fatal(err)
		}
	}
	defer tlsLn.Close()

	// Set up optimized HTTP transport
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		DisableKeepAlives:     false,
	}

	// Set up reverse proxy with optimizations
	rp := httputil.NewSingleHostReverseProxy(backendUrl)
	rp.Transport = transport
	rp.FlushInterval = 100 * time.Millisecond
	rp.BufferPool = &bufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 32*1024)
			},
		},
	}
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

	log.Printf("starting HTTPS reverse proxy to %s on port %d%s", cfg.Backend, cfg.HTTPSPort, logMess)
	log.Fatal(http.Serve(tlsLn, rp))
}
