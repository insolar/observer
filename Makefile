export GO111MODULE ?= on
export GOFLAGS ?= -mod=vendor

ARTIFACTS_DIR = .artifacts
BIN_DIR = bin
API = api
OBSERVER = observer
GOPATH ?= $(shell go env GOPATH)
PATH := $(GOPATH)/bin:$(PATH)

VERSION	:=
ifeq ($(OS),Windows_NT)
	VERSION := Windows
	ifeq ($(PROCESSOR_ARCHITECTURE),AMD64)
		VERSION := $(VERSION)_x86_64
	endif
	ifeq ($(PROCESSOR_ARCHITECTURE),x86)
		VERSION := $(VERSION)_i386
	endif
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		VERSION := Linux
	endif
	ifeq ($(UNAME_S),Darwin)
		VERSION := Darwin
	endif
	VERSION := $(VERSION)_$(shell uname -m)
endif

.PHONY: osflag
osflag:
	@echo $(VERSION)

.PHONY: build
build: ## build all binaries
	go build -o $(BIN_DIR)/$(OBSERVER) cmd/observer/*.go
	go build -o $(BIN_DIR)/$(API) cmd/api/*.go
	go build -o $(BIN_DIR)/stats-collector cmd/stats-collector/*.go
	go build -o $(BIN_DIR)/binance-collector cmd/binance-collector/*.go
	go build -o $(BIN_DIR)/migrate cmd/migrate/*.go
	go build -o $(BIN_DIR)/coin-market-cap-collector cmd/coin-market-cap-collector/*.go

.PHONY: install_deps
install_deps: minimock golangci

gobin: ## ensure gopath/bin
	mkdir -p ${GOPATH}/bin

.PHONY: minimock
minimock: gobin
	curl -sfL https://github.com/gojuno/minimock/releases/download/v3.0.6/minimock_3.0.6_${VERSION}.tar.gz | tar xzf - -C ${GOPATH}/bin/ minimock

golangci: $(BIN_DIR)
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${BIN_DIR} v1.21.0

.PHONY: generate
generate:
	go generate ./...

.PHONY: lint
lint: golangci
	${BIN_DIR}/golangci-lint --color=always run ./... -v --timeout 5m

$(BIN_DIR): ## create bin dir
	@mkdir -p $(BIN_DIR)

.PHONY: config
config: ## generate configs
	mkdir -p $(ARTIFACTS_DIR)
	go run ./configuration/gen/gen.go
	mv ./observer.yaml $(ARTIFACTS_DIR)/observer.yaml
	mv ./observerapi.yaml $(ARTIFACTS_DIR)/observerapi.yaml

ci_test: ## run tests with coverage
	go test -json -v -count 10 -timeout 20m --coverprofile=coverage.txt --covermode=atomic ./... | tee ci_test_with_coverage.json

.PHONY: test
test: ## tests
	go test ./... -v

integration: ## integration tests
	go test ./... -tags=integration -v

.PHONY: all
all: config build

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: migrate
migrate: ## migrate
	go run ./cmd/migrate/migrate.go --dir=scripts/migrations --init --config=.artifacts/observer.yaml
