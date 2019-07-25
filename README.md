# Repogirl
Mirrorlist that actively checks mirrors before serving them back. Written in Go.


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
Assuming you've built the container image with the name repogirl, starting it by hand with the proper environment variables looks like this:
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


# Enable TLS
If a `cert.pem` and `key.pem` file are present in the working directory (or bound in / of the container), repogirl will try to parse them and
if successful an extra HTTPS server will be started on port 8443 which supports TLS transport.

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
In case repogirl should double as an actual mirror, having a directory called `pub` present in the working directory will enable a fileserver
backend. All files and directories from `pub` (and only pub) will be served over HTTP (and if enabled over HTTPS).

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
  -v $PWD/pub:/pub
  repogirl
```
# Use

## Requesting a mirrorlist
Once up and running, repogirl will return a list of the mirrors that are reachable and have the specified repo and release. Optionally, depending on the mirror, an 'arch' parameter can be specified:
```
~$ curl -L 'http://localhost:8080/?repo=os&release=7&arch=x86_64'
http://centos.mirror.triple-it.nl/7/os/x86_64
http://mirror.dataone.nl/centos/7/os/x86_64
http://mirrors.xtom.nl/centos/7/os/x86_64
```

Of the 3 mirrors in this example, only one supports HTTPS. If all of them are specified having an 'https://' prefix in the REPO_MIRRORS variable, only the one actually supporting that is returned:
```
~$ curl -L 'http://localhost:8080/?repo=os&release=7&arch=x86_64'
https://mirrors.xtom.nl/centos/7/os/x86_64
```

## Requesting a repodiff
The output it trimmed for brevity. Requesting a repodiff between 2 existing releases of the same repo, repogirl will find which mirrors have the requested releases and do a repodiff between them. Caching the output should it be requested again.
```
http://localhost:8080/repodiff?old=previous&new=stable&repo=os&arch=x86_64
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

# Gotcha's

* Not setting any variables will cause repogirl to return 204's, there will be a warning about this
* Depending or the mirror's repo topology, some might need an additional 'arch' parameter
