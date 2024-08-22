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
	Port      int    `env:"PORT" envDefault:"443"`
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

	ln, err := tss.ListenTLS("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	rp := httputil.NewSingleHostReverseProxy(backendUrl)
	rp.ErrorHandler = func(
		w http.ResponseWriter,
		r *http.Request,
		err error,
	) {
		http.Error(w, "502 bad gateway", http.StatusBadGateway)
		log.Printf("backend error: %s", err)
	}

	log.Printf("start reverse proxy to %s", cfg.Backend)
	log.Fatal(http.Serve(ln, rp))
}
