PROJECT_DIR=$(shell pwd)


.DEFAULT_GOAL=install

.PHONY: release
export OMT_IMAGE_REPO ?= quay.io/operator-framework/opm
release: goreleaser
	goreleaser release --rm-dist

docs:
	mkdir -p docs && cd hack/build/docs && go run main.go

clean:
	rm -rf ./docs

test: ginkgo
	$(GINKGO) -r --randomizeAllSpecs --randomizeSuites --failOnPending --cover --trace --race --progress ./...

test-integration: tox install
	cd internal && tox -e integration

install:
	go install -ldflags='-X "github.com/operator-framework/operator-manifest-tools/cmd.Version=dev" -X "github.com/operator-framework/operator-manifest-tools/cmd.Commit=dev" -X "github.com/operator-framework/operator-manifest-tools/cmd.Date=$(shell date +"%Y-%m-%dT%H:%M:%S%z")"'

GINKGO=$(PROJECT_DIR)/bin/ginkgo
ginkgo:
	$(call go-get-tool,$(GINKGO),github.com/onsi/ginkgo/ginkgo)

tox:
	@[ -f $(command -v tox) ] || { \
	pip3 install tox ;\
	}

goreleaser:
	@[ -f $(which goreleaser) ] || go install github.com/goreleaser/goreleaser@latest

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo $(1) ;\
GOBIN=$(PROJECT_DIR)/bin go get -u $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
