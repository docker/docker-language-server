.PHONY: build
build:
	./build.sh

.PHONY: test
test:
	gotestsum -- $$(go list ./... | grep -v e2e-tests) -timeout 30s
	go test $$(go list ./... | grep e2e-tests) -timeout 240s

.PHONY: build-docker-test
build-docker-test:
	docker build -t docker/lsp:test --target test .

.PHONY: test-docker
test-docker: build-docker-test
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock docker/lsp:test make test

.PHONY: test-docker-disconnected
test-docker-disconnected: build-docker-test
	docker run -e DOCKER_NETWORK_NONE=true --rm -v /var/run/docker.sock:/var/run/docker.sock --network none docker/lsp:test make test

.PHONY: lint
lint:
	golangci-lint run

.PHONY: install
install:
	go install ./cmd/docker-language-server

.PHONY: vendor
vendor:
	docker buildx bake vendor

.PHONY: validate-vendor
validate-vendor:
	docker buildx bake vendor-validate

.PHONY: validate-all
validate-all: lint validate-vendor
