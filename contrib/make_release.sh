#!/usr/bin/env sh -eu

# usage: run this from the root of the repo
#   ./contrib/make_release.sh 2.1.0+dev 2.2.0

VERSION_FILE=version.go
NEW_VERSION_FILE=${VERSION_FILE}.new
OLD_TAG=$(echo "${1}" | sed "s/^/v/" | sed "s/+dev//")
NEW_TAG="v${2}"

sed "s/const Version = \"${1}\"/const Version = \"${2}\"/" ${VERSION_FILE} > ${NEW_VERSION_FILE}
mv ${NEW_VERSION_FILE} ${VERSION_FILE}
git add ${VERSION_FILE}
git commit -m "Release ${2}" -m "$(git shortlog ${OLD_TAG}..)"
git tag "${NEW_TAG}"

sed "s/const Version = \"${2}\"/const Version = \"${2}+dev\"/" ${VERSION_FILE} > ${NEW_VERSION_FILE}
mv ${NEW_VERSION_FILE} ${VERSION_FILE}
git add ${VERSION_FILE}
git commit -m "Start new development cycle"
echo "Now run"
echo "git push origin HEAD:refs/heads/master ${NEW_TAG}"
