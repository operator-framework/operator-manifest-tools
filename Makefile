PROJECT_DIR=$(shell pwd)


.DEFAULT_GOAL=install

.PHONY: release
export OMT_IMAGE_REPO ?= quay.io/operator-framework/operator-manifest-tools
release: goreleaser
	goreleaser release --rm-dist

docs:
	mkdir -p docs && cd hack/build/docs && go run main.go

clean:
	rm -rf ./docs

test: ginkgo
	$(GINKGO) -r --randomizeAllSpecs --randomizeSuites --failOnPending --cover --trace --race --progress ./...

test-integration: install
	cd internal && tox -e integration

install:
	go install -ldflags='-X "github.com/operator-framework/operator-manifest-tools/cmd.Version=dev" -X "github.com/operator-framework/operator-manifest-tools/cmd.Commit=dev" -X "github.com/operator-framework/operator-manifest-tools/cmd.Date=$(shell date +"%Y-%m-%dT%H:%M:%S%z")"'

GINKGO=$(PROJECT_DIR)/bin/ginkgo
LOCALBIN=$(PROJECT_DIR)/bin
ginkgo:
	GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/ginkgo@latest

goreleaser:
	@[ -f $(which goreleaser) ] || go install github.com/goreleaser/goreleaser@latest
