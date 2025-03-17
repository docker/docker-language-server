#!/bin/sh
VERSION="$(git describe --exact-match --tags 2> /dev/null)"
if [ $? -ne 0 ]; then
    VERSION="$(git show -s --format=%cs)-$(git rev-parse --short HEAD)"
fi

GOOS="${GOOS:=$(go env GOOS)}"
GOARCH="${GOARCH:=$(go env GOARCH)}"
OUTPUT="docker-language-server-${GOOS}-${GOARCH}"

# add .exe for Windows binaries
if [ "$GOOS" = "windows" ]; then
    OUTPUT="$OUTPUT.exe"
fi

echo "Building for ${GOOS}-${GOARCH}: ${OUTPUT}"

go build \
    -ldflags="-X 'github.com/docker/docker-language-server/internal/pkg/cli/metadata.Version=$VERSION'" \
    -o $OUTPUT \
    ./cmd/docker-language-server
