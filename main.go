package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thoas/stats"
	"gopkg.in/inconshreveable/log15.v2"
)

var (
	mirrors []string          // env REPO_MIRRORS="http://centos.mirror.triple-it.nl, http://mirror.dataone.nl/centos, http://mirrors.xtom.nl/centos"
	aliases map[string]string // env RELEASE_ALIASES="7=7.6.1810, 6=6.9"
	client  *http.Client

	version   = "0000000"
	buildtime = "0000000"
)

func init() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		DisableCompression: false,
	}
	client = &http.Client{Transport: tr}

	// parse repo mirrors from environment variable
	if mirrorsenv, ok := os.LookupEnv("REPO_MIRRORS"); !ok {
		warn("no repository mirrors specified in REPO_MIRRORS environment variable, replies will be status 204")
	} else {
		mirrors = strings.Split(mirrorsenv, ",")
		for i, m := range mirrors {
			mirrors[i] = strings.TrimSpace(m)
		}
	}

	// parse release aliases from environment variable
	aliases = make(map[string]string)
	if aliasesenv, ok := os.LookupEnv("RELEASE_ALIASES"); !ok {
		warn("no release aliases specified in RELEASE_ALIASES environment variable")
	} else {
		for _, a := range strings.Split(aliasesenv, ",") {
			a = strings.TrimSpace(a)
			p := strings.Split(a, "=")
			if len(p) != 2 {
				fatal("could not parse release alias", "failed", a)
			}
			aliases[p[0]] = p[1]
		}
	}

}

func fatal(msg string, ctx ...interface{}) {
	log15.Crit(msg, ctx...)
	os.Exit(1)
}

func info(msg string, ctx ...interface{}) {
	log15.Info(msg, ctx...)
}

func warn(msg string, ctx ...interface{}) {
	log15.Warn(msg, ctx...)
}

func debug(msg string, ctx ...interface{}) {
	log15.Debug(msg, ctx...)
}

func main() {

	// trap signals to properly shutdown http server
	signals := make(chan os.Signal, 1)
	defer close(signals)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// init built-in stats page (middleware will forward to muxer)
	middleware := stats.New()
	// setup actual muxer (muxer will handle all final requests)
	mux := http.NewServeMux()

	// requests for '/' should be parsed as a mirrorlist request
	mux.HandleFunc("/", mirrorsRequest)

	// requests for '/stats' should return relevant service stats
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(middleware.Data())
		w.Write(b)
	})

	// create channel for when http server gets shutdown for any reason
	shut := make(chan error)

	// init http server
	server := &http.Server{
		Addr:    ":8080",
		Handler: middleware.Handler(mux),
	}

	info("starting repogirl", "version", version, "buildtime", buildtime)
	// start the server in the background and pass the channel for error handling
	go func(srv *http.Server, err chan error) {
		err <- srv.ListenAndServe()
	}(server, shut)

	// while the http server is up and running
	for running := true; running; {
		select {
		case sig := <-signals:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				info("received signal", "signal", sig.String(), "action", "stopping")
				ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
				if err := server.Shutdown(ctx); err != nil {
					warn("http server exited without proper shutdown", "error", err.Error())
				}
			default:
				info("received signal", "signal", sig.String(), "action", "ignoring")
			}
		case err := <-shut:
			if err != http.ErrServerClosed {
				warn("http server exited without proper shutdown", "error", err.Error())
			}
			running = false
		}
	}
}
