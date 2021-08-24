PROJECT_DIR=$(shell pwd)

.DEFAULT_GOAL=install

release: goreleaser
	goreleaser release --rm-dist

docs:
	mkdir -p docs && cd hack/build/docs && go run main.go

clean:
	rm -rf ./docs

test: ginkgo
	$(GINKGO) -r --randomizeAllSpecs --randomizeSuites --failOnPending --cover --trace --race --progress ./...

install:
	go install

GINKGO=$(PROJECT_DIR)/bin/ginkgo
ginkgo:
	$(call go-get-tool,$(GINKGO),github.com/onsi/ginkgo/ginkgo)

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
