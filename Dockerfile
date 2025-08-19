# syntax=docker/dockerfile:1

ARG GO_VERSION="1.25"
ARG ALPINE_VERSION="3.22"

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS base
ENV CGO_ENABLED=0
RUN apk add --no-cache file git rsync make
WORKDIR /src

FROM base AS build-base
COPY go.* .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

FROM build-base AS vendored
RUN --mount=type=bind,target=.,rw \
    --mount=type=cache,target=/go/pkg/mod \
    go mod tidy && mkdir /out && cp go.mod go.sum /out

FROM scratch AS vendor-update
COPY --from=vendored /out /

FROM vendored AS vendor-validate
RUN --mount=type=bind,target=.,rw <<EOT
  set -e
  git add -A
  cp -rf /out/* .
  diff=$(git status --porcelain -- go.mod go.sum)
  if [ -n "$diff" ]; then
    echo >&2 'ERROR: Vendor result differs. Please vendor your package with "make vendor"'
    echo "$diff"
    exit 1
  fi
EOT

FROM base AS test
RUN <<SETUP
    apk add --no-cache make docker-cli && go install gotest.tools/gotestsum@latest
    wget https://github.com/docker/buildx/releases/download/v0.21.2/buildx-v0.21.2.linux-amd64
    chmod +x buildx-v0.21.2.linux-amd64
    mkdir ~/.docker/ ~/.docker/cli-plugins/
    mv buildx-v0.21.2.linux-amd64 ~/.docker/cli-plugins/docker-buildx
SETUP
COPY . .
# use -run=NON_EXISTING to download all the test dependencies
RUN make install && go test -run=NON_EXISTING ./...
