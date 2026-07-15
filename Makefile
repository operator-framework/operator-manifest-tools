.DEFAULT_GOAL=install

.PHONY: release
export OMT_IMAGE_REPO ?= quay.io/operator-framework/operator-manifest-tools
release: goreleaser
	goreleaser release --clean

docs:
	mkdir -p docs && cd hack/build/docs && go run main.go

clean:
	rm -rf ./docs

test: ginkgo
	$(GINKGO) -r --randomize-all --randomize-suites --fail-on-pending --cover --trace --race ./...

test-integration: install
	cd internal && tox -e integration

.PHONY: fmt
fmt: gofumpt golangci-lint
	${GOFUMPT} -l -w .
	$(GOLANGCI_LINT) fmt
	git diff --exit-code

.PHONY: tidy
tidy:
	go mod tidy
	git diff --exit-code

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter checks.
	$(GOLANGCI_LINT) run

install:
	go install -ldflags='-X "github.com/operator-framework/operator-manifest-tools/cmd.Version=dev" -X "github.com/operator-framework/operator-manifest-tools/cmd.Commit=dev" -X "github.com/operator-framework/operator-manifest-tools/cmd.Date=$(shell date +"%Y-%m-%dT%H:%M:%S%z")"'

GINKGO=$(PROJECT_DIR)/bin/ginkgo
LOCALBIN=$(PROJECT_DIR)/bin
ginkgo:
	GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo@latest

goreleaser:
	@[ -f $(which goreleaser) ] || go install github.com/goreleaser/goreleaser@latest

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.8.0
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT):
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION))

GOFUMPT = $(shell pwd)/bin/gofumpt
GOFUMPT_VERSION ?= v0.9.0
gofumpt: ## Download gofumpt locally if necessary.
	$(call go-install-tool,$(GOFUMPT),mvdan.cc/gofumpt@$(GOFUMPT_VERSION))

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-install-tool
@[ -f $(1) ] || { \
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
}
endef
