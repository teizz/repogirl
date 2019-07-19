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
    http://mirrors.xtom.nl/centos" \
  RELEASE_ALIASES=" \
    stable=7.6.1810, \
    stable6=6.9" \
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
       http://mirrors.xtom.nl/centos" \
  -e RELEASE_ALIASES=" \
       stable=7.6.1810, \
       stable6=6.9" \
  -p 8080:8080 \
  repogirl
```


# Enable TLS
If a `cert.pem` and `key.pem` file are present in the working directory (or bound in / of the container), repogirl will try to parse them and
if successful an extra HTTPS server will be started on port 8443 which supports TLS transport.


# Serving files
In case repogirl should double as an actual mirror, having a directory called `pub` present in the working directory will enable a fileserver
backend. All files and directories from `pub` (and only pub) will be served over HTTP and (if enabled over HTTPS).


# Use
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


# Gotcha's

* Not setting any variables will cause repogirl to return 204's, there will be a warning about this
* Depending or the mirror's repo topology, some might need an additional 'arch' parameter
