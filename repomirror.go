package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

func mirrorRepository(uri, repo string) (failed []string, err error) {
	if !checkMirror(uri) {
		err = fmt.Errorf("mirror for %s does not have valid metadata", repo)
		return
	}

	debug("repomirror", "status", "starting", "uri", uri)
	t0 := time.Now()

	if _, err = os.Stat("pub"); err != nil {
		err = fmt.Errorf("repomirror failed, 'pub' directory not set up correctly (%s)", err.Error())
		return
	}

	var c, f int
	var pkgsmd *pkgmd
	if pkgsmd, err = fetchPackageMetadata(uri); err != nil {
		err = fmt.Errorf("repomirror failed: %s", err.Error())
		return
	}

	// keep track of how many routines are running
	var running int64

	// create a channel for failure feedback from routines
	failchan := make(chan error)

	// separate routine for processing all failures and for each message
	// returned decrease the number of running routines by one
	go func() {
		for e := range failchan {
			if e != nil {
				debug("package download failed", "err", e.Error())
				failed = append(failed, e.Error())
				f++
			}
			c++
			atomic.AddInt64(&running, -1)
		}
	}()

	// create a routine for each package
	for _, p := range pkgsmd.Package {
		// sleep for a little bit while there are enough routines running
		for atomic.LoadInt64(&running) >= int64(fetchRoutines) {
			time.Sleep(time.Millisecond)
		}

		// increase the number of running routine by one and kick off a
		// new routine
		atomic.AddInt64(&running, 1)
		go func(u, r, h string, s int) {
			// arguments where: u = uri, r = localrepo, h = href, and s = package size
			tn := time.Now()

			// the href will have to be copied with the correct path, so split the
			// package name and package path the package can be kept in the same
			// relative path.
			pkgcomponents := strings.Split(h, "/")
			pkgname := pkgcomponents[len(pkgcomponents)-1]
			pkgcomponents = pkgcomponents[:len(pkgcomponents)-1]

			// calculate where to local package should go, which is the localrepo name
			// combined with the package href converted to the correct path notation for
			// the OS this instance is running on.
			repocomponents := append([]string{"pub"}, strings.Split(r, "/")...)

			// append the package components to the repocomponents so relative the the
			// localrepo the href of the package would still be correct.
			repocomponents = append(repocomponents, pkgcomponents...)

			// finally create a string with the full path the individual package should
			// be downloaded to.
			pkgpath := path.Join(repocomponents...)

			if fd, err := os.Stat(path.Join(pkgpath, pkgname)); err == nil {
				if int64(s) == fd.Size() {
					debug("repomirror", "status", "already present", "package", pkgname)
					failchan <- nil
					return
				}
				warn("repomirror", "status", "incorrect size", "package", pkgname)
			}

			if err := os.MkdirAll(pkgpath, 0755); err != nil {
				err = fmt.Errorf("unable to create directory for %s (%s)", pkgname, err.Error())
				failchan <- err
			} else {
				var resp *http.Response
				if resp, err = client.Get(u + "/" + h); err != nil {
					err = fmt.Errorf("unable to download package %s (%s)", pkgname, err.Error())
					failchan <- err
				} else {
					defer resp.Body.Close()
					if fh, err := os.OpenFile(path.Join(pkgpath, pkgname), os.O_RDWR|os.O_CREATE, 0644); err != nil {
						err = fmt.Errorf("unable to write %s (%s)", pkgname, err.Error())
						failchan <- err
					} else {
						var sz int64
						if sz, err = io.Copy(fh, resp.Body); err != nil {
							err = fmt.Errorf("failure writing package %s (%s)", pkgname, err.Error())
							failchan <- err
						} else if sz != int64(s) {
							err = fmt.Errorf("written size does not match expected size for %s", pkgname)
							failchan <- err
						} else {
							speed := float64(s) / 1024 / time.Since(tn).Seconds()
							debug("repomirror", "status", "downloaded", "package", pkgname, "size", fmt.Sprintf("%.2fKB", float64(s)/1024), "speed", fmt.Sprintf("%.2fKB/s", speed))
							failchan <- nil
						}
					}

				}
			}
		}(uri, repo, p.Location.Href, p.Size.Package)
	}
	// while there are still routines running, take a little nap
	for atomic.LoadInt64(&running) > 0 {
		time.Sleep(time.Millisecond)
	}

	// be nice and close the channel so the error reporting routine
	// can stop as well
	close(failchan)

	debug("repomirror", "status", "done", "uri", uri, "repo", repo, "total", c, "failed", f, "elapsed", time.Since(t0))

	if c < 1 {
		err = fmt.Errorf("no packages checked for %s", uri)
	}
	return
}

func mirrorRequest(w http.ResponseWriter, r *http.Request) {
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

			localrepo := release + "/" + repo
			if len(arch) > 0 {
				localrepo += "/" + arch
			}

			var failed []string
			var err error
			if failed, err = mirrorRepository(uri, localrepo); err != nil {
				warn("unable to mirror repo", "mirror", mirror, "release", release, "repo", repo, "err", err.Error())
				w.Write([]byte(uri + " NOT MIRRORED\n"))
			} else if len(failed) > 0 {
				warn("some packages not mirrored", "mirror", mirror, "release", release, "repo", repo, "failed", len(failed))
				w.Write([]byte(uri + " " + strconv.Itoa(len(failed)) + " FAILED PACKAGES\n"))
			} else {
				info("all packages mirrored successfully", "mirror", mirror, "release", release, "repo", repo)
				w.Write([]byte(uri + " OK\n"))
			}
		}
	}
}
