# repogirl
Mirrorlist that actively checks mirrors before serving them back. Written in Go.

# Running after building

* Set environment variables a list of mirrors you assume will be up (comma separated).
* Optionally set some aliases for releases you want to dynamically re-point (comma separated, int the form of name=destination).

for example:
```
env REPO_MIRRORS="http://centos.mirror.triple-it.nl, http://mirror.dataone.nl/centos, http://mirrors.xtom.nl/centos" \
  RELEASE_ALIASES="stable=7.6.1810, stable6=6.9" \
  ./repogirl
```

# Running in a container
Assuming you've built the container image with the name repogirl, starting it by hand with the proper environment variables looks like this:
```
docker container run \
  --rm \
  -e REPO_MIRRORS="http://centos.mirror.triple-it.nl, http://mirror.dataone.nl/centos, http://mirrors.xtom.nl/centos" \
  -e RELEASE_ALIASES="stable=7.6.1810, stable6=6.9" \
  -p 8080:8080 \
  repogirl
```

# Gotcha's

* Not setting any variables will cause repogirl to return 204's, there will be a warning about this
