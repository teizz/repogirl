# Repogirl
Mirrorlist that actively checks mirrors before serving them back. Written in Go.


# Features
* Config via environment variables:
    * Quick check of uri's in `REPO_MIRRORS` for an existing release of a requested repo before serving.
    * On-the-fly aliasing of releases to easy to remember names using `RELEASE_ALIASES`.
    * Disable checking of mirror TLS certificates by setting `INSECURE_SKIP_VERIFY=1`.
    * HTTP client will attempt to use `HTTP_PROXY` for proxy if defined.
    * Extra debugging information reported if `DEBUG` is set to anything other than '0', 'no', or 'false'.
    * Parallel checking of repohealth and mirroring with repomirror configurable as `FETCH_ROUTINES=16` (default).
* Opportunistic HTTPS support if a `key.pem` and `cert.pem` are found.
* Opportunistic HTTPS-client support if a `client-key.pem` and `client-cert.pem` are found.
* Opportunistic serving of files from a directory named `pub` if found.

* On-demand repo diffs between 2 releases (possibly from different mirrors).
* On-demand repo health check of all mirrors (checks reported package size against metadata).
* On-demand repo mirror which downloads all packages from all mirrors unless already present.

* Built-in server stats served from `/stats` in JSON format.
* Possible to build into a single binary (+CA-certs) container.


# Running after building

* Set environment variables a list of mirrors you assume will be up (comma separated).
* Optionally set some aliases for releases you want to dynamically re-point (comma separated, int the form of name=destination).

for example:
```
env REPO_MIRRORS=" \
    http://centos.mirror.triple-it.nl, \
    http://mirror.dataone.nl/centos, \
    http://mirrors.xtom.nl/centos, \
    http://vault.centos.org" \
  RELEASE_ALIASES=" \
    stable=7.6.1810, \
    previous=7.5.1804" \
  ./repogirl
```


# Running in a container
Pulling the container image from [mattijs/repogirl](https://hub.docker.com/r/mattijs/repogirl),
starting it by hand with the proper environment variables looks like this (note how
it will find `stable` on any of the current mirrors and `previous` on vault):
```
docker container run \
  --rm \
  -e REPO_MIRRORS=" \
       http://centos.mirror.triple-it.nl, \
       http://mirror.dataone.nl/centos, \
       http://mirrors.xtom.nl/centos, \
       http://vault.centos.org" \
  -e RELEASE_ALIASES=" \
       stable=7.6.1810, \
       previous=7.5.1804" \
  -p 8080:8080 \
  repogirl
```

# Disable TLS verification
Should mirrors be serving repos over HTTPS but with a certificate that cannot be
verified by the default CA chain, then it is possible to disable this
verification.

Setting the environment variable `INSECURE_SKIP_VERIFY=1` disables TLS verification.

# Enable TLS
If a `cert.pem` and `key.pem` file are present in the working directory (or
bound in `/` of the container), repogirl will try to parse them and if successful
an extra HTTPS server will be started on port 8443 which supports TLS transport.

## Example
```
docker container run \
  --rm \
  -e REPO_MIRRORS=" \
       http://centos.mirror.triple-it.nl, \
       http://mirror.dataone.nl/centos, \
       http://mirrors.xtom.nl/centos, \
       http://vault.centos.org" \
  -p 8080:8080 \
  -v $PWD/cert.pem:/cert.pem \
  -v $PWD/key.pem:/key.pem \
  repogirl
```


# Serving files
In case repogirl should double as an actual mirror, having a directory called `pub`
present in the working directory will enable a fileserver backend. All files and
directories from `pub` (and only pub) will be served over HTTP (and if enabled over HTTPS).

## Example
```
docker container run \
  --rm \
  -e REPO_MIRRORS=" \
       http://centos.mirror.triple-it.nl, \
       http://mirror.dataone.nl/centos, \
       http://mirrors.xtom.nl/centos, \
       http://vault.centos.org" \
  -p 8080:8080 \
  -v $PWD/pub:/pub \
  repogirl
```


# Use

## Requesting a mirrorlist ('/' or '/mirrorlist')
Once up and running, repogirl will return a list of the mirrors that are reachable and have the specified repo and release. Optionally, depending on the mirror, an 'arch' parameter can be specified:
```
~$ curl -L 'http://localhost:8080/?repo=os&release=7&arch=x86_64'
http://centos.mirror.triple-it.nl/7/os/x86_64
http://mirror.dataone.nl/centos/7/os/x86_64
http://mirrors.xtom.nl/centos/7/os/x86_64
```

Of the 3 mirrors in this example, only one supports HTTPS. If all of them are specified having an 'https://' prefix in the `REPO_MIRRORS` variable, only the one actually supporting that is returned:
```
~$ curl -L 'http://localhost:8080/?repo=os&release=7&arch=x86_64'
https://mirrors.xtom.nl/centos/7/os/x86_64
```

## Requesting a repodiff ('/repodiff')
The output it trimmed for brevity. Requesting a repodiff between 2 existing releases of the same repo, repogirl will find which mirrors have the requested releases and do a repodiff between them. Caching the output should it be requested again.
```
~> curl -L 'http://localhost:8080/repodiff?old=previous&new=stable&repo=os&arch=x86_64'
added:
	bcc-0.6.1-2.el7.x86_64
	bcc-devel-0.6.1-2.el7.x86_64
	bcc-doc-0.6.1-2.el7.noarch
	...
	...
changed:
	389-ds-base-1.3.7.5-18.el7.x86_64 -> 389-ds-base-1.3.8.4-15.el7.x86_64
	389-ds-base-devel-1.3.7.5-18.el7.x86_64 -> 389-ds-base-devel-1.3.8.4-15.el7.x86_64
	389-ds-base-libs-1.3.7.5-18.el7.x86_64 -> 389-ds-base-libs-1.3.8.4-15.el7.x86_64
	...
	...
removed:
	cheese-camera-service-3.22.1-2.el7.x86_64
	evolution-mapi-devel-3.22.6-1.el7.i686
	evolution-mapi-devel-3.22.6-1.el7.x86_64
	...
	...
```

## Requesting a healthcheck ('/repohealth')
This will attempt to fetch the metadata of all mirrors, and check the reported package size from the HTTP headers with the size reported in the metadata. This **will** fetch all headers, using multiple threads so mirrors might not like this behaviour.
```
~> curl -L 'http://localhost:8080/repohealth?repo=extras&release=7&arch=x86_64'
http://centos.mirror.triple-it.nl/7/extras/x86_64 OK
http://mirror.dataone.nl/centos/7/extras/x86_64 OK
http://mirrors.xtom.nl/centos/7/extras/x86_64 OK
http://vault.centos.org/7/extras/x86_64 NOT CHECKED
```

## Requesting a repomirror ('/repomirror')
All packages for all available  mirrors will be checked for size and downloaded to the correct path if a `pub` directory is available. If a package is already present and has the correct size, it will be skipped.
```
~> curl 'http://localhost:8080/repomirror?repo=extras&release=7&arch=x86_64'
http://centos.mirror.triple-it.nl/7/extras/x86_64 OK
http://mirror.dataone.nl/centos/7/extras/x86_64 OK
http://mirrors.xtom.nl/centos/7/extras/x86_64 OK
http://vault.centos.org/7/extras/x86_64 NOT MIRRORED
```
After this, packages are found (and served right away) in `/pub/7/extras/x86_64` placed in `Packages` since that's where the metadata pointed to. Future plans include mirroring the metadata too so a full mirror is established out-of-the-box.

# Gotcha's

* Not setting any mirror variables will cause repogirl to return 204's when requesting
  a mirrorlist. There will be a warning about this.
* Depending or the mirror's repo topology, some might need an additional 'arch' parameter.
