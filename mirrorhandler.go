package main

import (
	"context"
	"net/http"
	"sync"
	"time"
)

var (
	mirrorcache = &sync.Map{}
)

type repomirror struct {
	uri       string
	valid     bool
	lastcheck time.Time
}

func checkMirror(uri string) (success bool) {
	var req *http.Request
	var resp *http.Response
	var err error
	var m repomirror

	if v, found := mirrorcache.Load(uri); !found {
		m = repomirror{uri: uri}
	} else {
		m = v.(repomirror)
	}

	if time.Since(m.lastcheck) > time.Minute {
		// log a debug line to show caching effect in action
		debug("updating mirror status", "uri", uri, "last check", m.lastcheck.Round(time.Second))

		// assume valid replies answer within 2 seconds or they are to slow, add
		// a timeout to the request so it will fail if not completed within the
		// timeout
		ctx, _ := context.WithTimeout(context.Background(), time.Second*2)

		// do not cache the request if for some reason a request could not be built
		if req, err = http.NewRequest("GET", uri+"/repodata/", nil); err != nil {
			warn("unable to build http request", "uri", uri)
			return
		}

		// if the client returns with an error (like invalid TLS certificates)
		// then do cache that result.
		if resp, err = client.Do(req.WithContext(ctx)); err != nil {
			warn("http client returned an error", "uri", req.RequestURI, "error", err)
			m.valid = false
		} else if resp.StatusCode != http.StatusOK {
			// if the statuscode is anything else than OK, also cache negatively
			m.valid = false
		} else {
			// only in case of statuscode being okay should cache be positive
			m.valid = true
		}
		m.lastcheck = time.Now()
		mirrorcache.Store(uri, m)
	}

	success = m.valid
	return
}

func mirrorsRequest(w http.ResponseWriter, r *http.Request) {
	release := r.URL.Query().Get("release")
	repo := r.URL.Query().Get("repo")
	arch := r.URL.Query().Get("arch")

	if len(release) < 1 || len(repo) < 1 {
		warn("not enough parameters sent", "uri", r.RequestURI, "release", release, "repo", repo)
		w.WriteHeader(http.StatusBadRequest)
	} else {
		if alias, ok := aliases[release]; ok {
			release = alias
		}

		var resp string
		var count int
		for _, mirror := range mirrors {
			uri := mirror + "/" + release + "/" + repo
			if len(arch) > 0 {
				uri += "/" + arch
			}
			if checkMirror(uri) {
				resp += uri + "\n"
				count++
			} else {
				warn("mirror does not have requested repo", "mirror", mirror, "release", release, "repo", repo)
			}
		}

		if count > 0 {
			debug("sending mirrors", "client", r.RemoteAddr, "up", count, "repo", repo, "release", r.URL.Query().Get("release"), "alias", release)
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Cache-Control", "max-age=3600")
			w.Write([]byte(resp))
		} else {
			warn("no mirrors sent", "client", r.RemoteAddr, "repo", repo, "release", r.URL.Query().Get("release"), "alias", release)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}
