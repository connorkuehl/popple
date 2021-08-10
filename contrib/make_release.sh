#!/usr/bin/env sh

set -exu

# usage: run this from the root of the repo
#   ./contrib/make_release.sh 2.1.0+dev 2.2.0

FROM="${1}"
TO="${2}"
LAST_TAG="$(git describe --tags --abbrev=0)"

sed -i "s/VERSION := ${FROM}/VERSION := ${TO}/" Makefile
git add Makefile
git commit -m "Release ${TO}" -m "$(git shortlog "${LAST_TAG}"..)"
git tag "${TO}"

sed -i "s/VERSION := ${TO}/VERSION := ${TO}+dev/" Makefile
git add Makefile
git commit -m "Start new development cycle"
echo "Now run: git push origin HEAD:refs/heads/master ${TO}"
