package main

import (
	"bufio"
	"compress/gzip"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	diffcache = &sync.Map{}
)

type repomd struct {
	XMLName  xml.Name `xml:"repomd"`
	Xmlns    string   `xml:"xmlns,attr"`
	Rpm      string   `xml:"rpm,attr"`
	Revision string   `xml:"revision"`
	Data     []struct {
		Type     string `xml:"type,attr"`
		Checksum struct {
			Text string `xml:",chardata"`
			Type string `xml:"type,attr"`
		} `xml:"checksum"`
		Location struct {
			Href string `xml:"href,attr"`
		} `xml:"location"`
		Timestamp    string `xml:"timestamp"`
		Size         string `xml:"size"`
		OpenChecksum struct {
			Text string `xml:",chardata"`
			Type string `xml:"type,attr"`
		} `xml:"open-checksum"`
		OpenSize        string `xml:"open-size"`
		DatabaseVersion string `xml:"database_version"`
	} `xml:"data"`
}

type pkgmd struct {
	XMLName  xml.Name `xml:"metadata"`
	Xmlns    string   `xml:"xmlns,attr"`
	Rpm      string   `xml:"rpm,attr"`
	Packages string   `xml:"packages,attr"` // number of packages
	Package  []struct {
		Text    string `xml:",chardata"`
		Type    string `xml:"type,attr"`
		Name    string `xml:"name"`
		Arch    string `xml:"arch"`
		Version struct {
			Epoch string `xml:"epoch,attr"`
			Ver   string `xml:"ver,attr"`
			Rel   string `xml:"rel,attr"`
		} `xml:"version"`
		Checksum struct {
			Text  string `xml:",chardata"`
			Type  string `xml:"type,attr"`
			Pkgid string `xml:"pkgid,attr"`
		} `xml:"checksum"`
		Summary     string `xml:"summary"`
		Description string `xml:"description"`
		Packager    string `xml:"packager"`
		URL         string `xml:"url"`
		Time        struct {
			File  int `xml:"file,attr"`
			Build int `xml:"build,attr"`
		} `xml:"time"`
		Size struct {
			Package   int `xml:"package,attr"`
			Installed int `xml:"installed,attr"`
			Archive   int `xml:"archive,attr"`
		} `xml:"size"`
		Location struct {
			Href string `xml:"href,attr"`
		} `xml:"location"`
	} `xml:"package"`
}

type pkgshort struct {
	name string
	arch string
}

type pkgvers struct {
	ver  string
	rel  string
	time int
}

type repodiff struct {
	lastcheck time.Time
	added     []string
	changed   []string
	removed   []string
}

func fetchFileLists(uri string) (result map[pkgshort]pkgvers) {
	rmd := &repomd{}
	if resp, err := client.Get(uri + "/repodata/repomd.xml"); err != nil {
		fatal("unable to fetch repomd.xml", "uri", uri, "err", err.Error())
	} else {
		if data, _ := ioutil.ReadAll(resp.Body); err != nil {
			fatal("unable to decode xml", "err", err.Error())
		} else {
			xml.Unmarshal(data, &rmd)
		}
	}

	result = make(map[pkgshort]pkgvers)

	pkgsmd := &pkgmd{}
	for _, d := range rmd.Data {
		if d.Type == "primary" {
			if resp, err := client.Get(uri + "/" + strings.TrimLeft(d.Location.Href, "/")); err != nil {
				fatal("unable to fetch filelist", "uri", uri, "err", err.Error())
			} else {
				body := bufio.NewReaderSize(resp.Body, 16*1024*1024)
				if respzip, err := gzip.NewReader(body); err != nil {
					fatal("unable to read gzip", "err", err.Error())
				} else if data, err := ioutil.ReadAll(respzip); err != nil {
					fatal("unable to read filelist", "err", err.Error())
				} else {
					xml.Unmarshal(data, &pkgsmd)
					for _, p := range pkgsmd.Package {
						entry := pkgshort{name: p.Name, arch: p.Arch}
						if first, dup := result[entry]; dup {
							if p.Time.File > first.time {
								debug("superceded package information", "package", p.Name, "version-1", first.ver+"-"+first.rel, "version-2", p.Version.Ver+"-"+p.Version.Rel, "time-1", first.time, "time-2", p.Time.File)
								result[entry] = pkgvers{ver: p.Version.Ver, rel: p.Version.Rel, time: p.Time.File}
							} else {
								debug("older package information", "package", p.Name, "version-1", first.ver+"-"+first.rel, "version-2", p.Version.Ver+"-"+p.Version.Rel, "time-1", first.time, "time-2", p.Time.File)
							}
						} else {
							result[entry] = pkgvers{ver: p.Version.Ver, rel: p.Version.Rel, time: p.Time.File}
						}
					}
				}
			}
		}
	}
	return
}

func mirrordiff(releaseold, releasenew string) (added, changed, removed []string) {
	pkgold := fetchFileLists(releaseold)
	pkgnew := fetchFileLists(releasenew)

	changed = make([]string, 0)
	for p, newvers := range pkgnew {
		if oldvers, found := pkgold[p]; found {
			if oldvers != newvers {
				changed = append(changed, p.name+"-"+oldvers.ver+"-"+oldvers.rel+"."+p.arch+" -> "+p.name+"-"+newvers.ver+"-"+newvers.rel+"."+p.arch)
			}
			delete(pkgnew, p)
			delete(pkgold, p)
		}
	}

	added = make([]string, 0)
	for p, newvers := range pkgnew {
		added = append(added, p.name+"-"+newvers.ver+"-"+newvers.rel+"."+p.arch)
	}

	removed = make([]string, 0)
	for p, oldvers := range pkgold {
		removed = append(removed, p.name+"-"+oldvers.ver+"-"+oldvers.rel+"."+p.arch)
	}

	sort.Strings(added)
	sort.Strings(changed)
	sort.Strings(removed)
	return
}

func diffRequest(w http.ResponseWriter, r *http.Request) {
	releaseold := r.URL.Query().Get("old")
	releasenew := r.URL.Query().Get("new")
	repo := r.URL.Query().Get("repo")
	arch := r.URL.Query().Get("arch")
	mirrorsold := make([]string, 0)
	mirrorsnew := make([]string, 0)

	if len(releaseold) < 1 || len(releasenew) < 1 || len(repo) < 1 {
		warn("not enough parameters sent", "uri", r.RequestURI, "old", releaseold, "new", releasenew, "repo", repo)
		w.WriteHeader(http.StatusBadRequest)
	} else if len(mirrors) < 1 {
		w.WriteHeader(http.StatusNoContent)
	} else {
		if alias, ok := aliases[releaseold]; ok {
			releaseold = alias
		}
		if alias, ok := aliases[releasenew]; ok {
			releasenew = alias
		}

		var diff repodiff
		if tdiff, found := diffcache.Load(releaseold + releasenew + repo + arch); found {
			diff = tdiff.(repodiff)
		} else {
			for _, mirror := range mirrors {
				uri := mirror + "/" + releaseold + "/" + repo
				if len(arch) > 0 {
					uri += "/" + arch
				}
				if checkMirror(uri) {
					mirrorsold = append(mirrorsold, uri)
				} else {
					warn("mirror does not have requested repo", "mirror", mirror, "release", releaseold, "repo", repo)
				}
			}

			if len(mirrorsold) > 0 {
				for _, mirror := range mirrors {
					uri := mirror + "/" + releasenew + "/" + repo
					if len(arch) > 0 {
						uri += "/" + arch
					}
					if checkMirror(uri) {
						mirrorsnew = append(mirrorsnew, uri)
					} else {
						warn("mirror does not have requested repo", "mirror", mirror, "release", releasenew, "repo", repo)
					}
				}
			}

			// mirrorsnew only gets filled if mirrorsold had at least one mirror
			// so it's safe to check just the mirrorsnew
			if len(mirrorsnew) > 0 {
				debug("diffing packages",
					"client", r.RemoteAddr,
					"repo", repo,
					"old", r.URL.Query().Get("old"),
					"alias", releaseold,
					"new", r.URL.Query().Get("new"),
					"alias", releasenew,
				)
				diff = repodiff{lastcheck: time.Now()}
				diff.added, diff.changed, diff.removed = mirrordiff(mirrorsold[0], mirrorsnew[0])
				diffcache.Store(releaseold+releasenew+repo+arch, diff)
			}
		}

		if !diff.lastcheck.IsZero() {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Cache-Control", "max-age=86400")
			w.Header().Set("X-Content-Age", time.Since(diff.lastcheck).Round(time.Second).String())
			w.WriteHeader(http.StatusOK)
			if len(diff.added)+len(diff.changed)+len(diff.removed) > 0 {
				w.Write([]byte("added:\n\t" + strings.Join(diff.added, "\n\t") + "\n"))
				w.Write([]byte("changed:\n\t" + strings.Join(diff.changed, "\n\t") + "\n"))
				w.Write([]byte("removed:\n\t" + strings.Join(diff.removed, "\n\t") + "\n"))
			} else {
				w.Write([]byte("no changes in packages\n"))
			}
		} else {
			warn("not enough mirrors to diff packages",
				"client", r.RemoteAddr,
				"repo", repo,
				"old release", releaseold,
				"old mirrors", len(mirrorsold),
				"new release", releasenew,
				"new mirrors", len(mirrorsnew),
			)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}
