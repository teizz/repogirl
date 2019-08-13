package main

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func checkHealth(uri string) (failed []string, err error) {
	var resp *http.Response
	if resp, err = client.Get(uri + "/repodata/repomd.xml"); err != nil {
		err = fmt.Errorf("unable to fetch repomd.xml (%s)", err.Error())
		return
	}
	defer resp.Body.Close()

	var rmd *repomd
	if err = xml.NewDecoder(resp.Body).Decode(&rmd); err != nil {
		err = fmt.Errorf("unable to read repomd.xml (%s)", err.Error())
		return
	}

	var c, f int
	for _, d := range rmd.Data {
		if d.Type == "primary" {
			// fetch the primary data from the repo
			var resp *http.Response
			if resp, err = client.Get(uri + "/" + strings.TrimLeft(d.Location.Href, "/")); err != nil {
				err = fmt.Errorf("unable to fetch filelist (%s)", err.Error())
				return
			}
			defer resp.Body.Close()

			// tie the gzipped data to a gzip.Reader
			var respzip io.Reader
			if respzip, err = gzip.NewReader(resp.Body); err != nil {
				err = fmt.Errorf("unable to decompress primary.xml.gz (%s)", err.Error())
				return
			}

			var pkgsmd *pkgmd
			if err = xml.NewDecoder(respzip).Decode(&pkgsmd); err != nil {
				err = fmt.Errorf("unable to read filelist (%s)", err.Error())
				return
			}

			for _, p := range pkgsmd.Package {
				c++
				if err = verifyContentSize(uri+"/"+p.Location.Href, p.Size.Package); err != nil {
					failed = append(failed, p.Location.Href+" "+err.Error())
					f++
				}
			}
		}
	}
	debug("packages checked", "uri", uri, "total", c, "failed", f)

	if c < 1 {
		err = fmt.Errorf("no packages checked for %s", uri)
	}
	return
}

func verifyContentSize(uri string, size int) (err error) {
	var resp *http.Response
	if resp, err = client.Head(uri); err != nil {
		err = fmt.Errorf("unable to fetch headers (%s)", err.Error())
	} else {
		if resp.ContentLength != int64(size) {
			err = fmt.Errorf("%d != %d", resp.ContentLength, size)
		}
	}
	return
}

func healthRequest(w http.ResponseWriter, r *http.Request) {
	release := r.URL.Query().Get("release")
	repo := r.URL.Query().Get("repo")
	arch := r.URL.Query().Get("arch")

	if len(release) < 1 || len(repo) < 1 {
		warn("not enough parameters sent", "release", release, "repo", repo, "uri", r.RequestURI)
		w.WriteHeader(http.StatusBadRequest)
	} else if len(mirrors) < 1 {
		w.WriteHeader(http.StatusNoContent)
	} else {
		if alias, ok := aliases[release]; ok {
			release = alias
		}

		for _, mirror := range mirrors {
			uri := mirror + "/" + release + "/" + repo
			if len(arch) > 0 {
				uri += "/" + arch
			}

			var failed []string
			var err error
			if failed, err = checkHealth(uri); err != nil {
				warn("unable to check health", "mirror", mirror, "release", release, "repo", repo, "err", err.Error())
				w.Write([]byte(uri + " NOT CHECKED\n"))
			} else if len(failed) > 0 {
				warn("some packages failed check", "mirror", mirror, "release", release, "repo", repo, "failed", len(failed))
				w.Write([]byte(uri + " " + strconv.Itoa(len(failed)) + " FAILED PACKAGES\n"))
			} else {
				info("all packages verified successfully", "mirror", mirror, "release", release, "repo", repo)
				w.Write([]byte(uri + " OK\n"))
			}
		}
	}
}
