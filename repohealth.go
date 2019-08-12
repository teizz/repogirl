package main

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

func checkHealth(uri string) (failed []string, err error) {
	rmd := &repomd{}
	var resp *http.Response
	if resp, err = client.Get(uri + "/repodata/repomd.xml"); err != nil {
		fatal("unable to fetch repomd.xml", "uri", uri, "err", err.Error())
	} else {
		if data, _ := ioutil.ReadAll(resp.Body); err != nil {
			fatal("unable to decode xml", "err", err.Error())
		} else {
			xml.Unmarshal(data, &rmd)
		}
	}

	pkgsmd := &pkgmd{}
	for _, d := range rmd.Data {
		if d.Type == "primary" {
			// fetch the primary data from the repo
			var resp *http.Response
			if resp, err = client.Get(uri + "/" + strings.TrimLeft(d.Location.Href, "/")); err != nil {
				err = fmt.Errorf("unable to fetch filelist (%s)", err.Error())
				return
			}

			// tie the gzipped data to a gzip.Reader
			var respzip io.Reader
			if respzip, err = gzip.NewReader(resp.Body); err != nil {
				err = fmt.Errorf("unable to read primary.xml.gz (%s)", err.Error())
				return
			}
			defer resp.Body.Close()

			// finally unzip the data into memory
			var data []byte
			if data, err = ioutil.ReadAll(respzip); err != nil {
				err = fmt.Errorf("unable to read filelist (%s)", err.Error())
				return
			}

			xml.Unmarshal(data, &pkgsmd)
			for _, p := range pkgsmd.Package {
				if err = verifyContentSize(uri+"/"+p.Location.Href, p.Size.Package); err != nil {
					failed = append(failed, p.Location.Href+" "+err.Error())
				}
			}
		}
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

		checkGood := []string{}
		checkFail := []string{}
		checkMiss := []string{}

		for _, mirror := range mirrors {
			uri := mirror + "/" + release + "/" + repo
			if len(arch) > 0 {
				uri += "/" + arch
			}

			var failed []string
			var err error
			if failed, err = checkHealth(uri); err != nil {
				warn("unable to check health", "mirror", mirror, "release", release, "repo", repo, "err", err.Error())
				checkMiss = append(checkMiss, fmt.Sprintf("%s NOT CHECKED", mirror))
			}

			if len(failed) > 0 {
				warn("some packages failed check", "mirror", mirror, "release", release, "repo", repo, "failed", len(failed))
				checkFail = append(checkFail, fmt.Sprintf("%s %d PACKAGES FAILED", mirror, len(failed)))
				return
			}

			info("all packages verified successfully", "mirror", mirror, "release", release, "repo", repo)
			checkGood = append(checkGood, fmt.Sprintf("%s OK", mirror))
		}

		w.WriteHeader(http.StatusOK)
		if len(checkMiss) > 0 {
			w.Write([]byte(strings.Join(checkMiss, "\n") + "\n"))
		}
		if len(checkFail) > 0 {
			w.Write([]byte(strings.Join(checkFail, "\n") + "\n"))
		}
		if len(checkGood) > 0 {
			w.Write([]byte(strings.Join(checkGood, "\n") + "\n"))
		}
	}
}
