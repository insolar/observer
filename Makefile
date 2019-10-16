BIN_DIR = bin
OBSERVER = observer
ARTIFACTS = .artifacts
CONFIG = config

.PHONY: build
build: $(BIN_DIR) $(OBSERVER) ## build!

.PHONY: env
env: $(ARTIFACTS) $(CONFIG) ## gen config + artifacts

$(BIN_DIR): ## create bin dir
	mkdir -p $(BIN_DIR)

.PHONY: $(OBSERVER)
$(OBSERVER):
	go build -o $(BIN_DIR)/$(OBSERVER) cmd/observer/*.go

$(ARTIFACTS):
	mkdir -p $(ARTIFACTS)

.PHONY: $(CONFIG)
$(CONFIG):
	go run ./configuration/gen/gen.go
	mv ./observer.yaml $(ARTIFACTS)/observer.yaml

.PHONY: ensure
ensure: ## dep ensure
	dep ensure -v

ci_test: ## run tests with coverage
	go test -json -v -count 10 -timeout 20m --coverprofile=converage.txt --covermode=atomic ./... | tee ci_test_with_coverage.json

.PHONY: test
test:
	go test ./...

integration:
	go test ./... -tags=integration

.PHONY: all
all: ensure env build ## ensure + build + artifacts

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
