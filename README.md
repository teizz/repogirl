# repogirl
Mirrorlist that actively checks mirrors before serving them back. Written in Go.

# start

* Set environment variables a list of mirrors you assume will be up (comma separated).
* Optionally set some aliases for releases you want to dynamically re-point (comma separated, int the form of name=destination).

for example:
env REPO_MIRRORS="http://centos.mirror.triple-it.nl, http://mirror.dataone.nl/centos, http://mirrors.xtom.nl/centos" RELEASE_ALIASES="stable=7.6.1810, stable6=6.9" ./repogirl

