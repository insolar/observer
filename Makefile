ARTIFACTS = .artifacts
BIN_DIR = bin
OBSERVER = observer
CONFIG = config
GOPATH ?= $(shell go env GOPATH)
PATH := $(GOPATH)/bin:$(PATH)
GO111MODULE := on

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
build: $(BIN_DIR) $(OBSERVER) api xns_stats_count ## build!

.PHONY: env
env: $(CONFIG) ## gen configs + artifacts

.PHONY: install_deps
install_deps: minimock golangci

gobin: ## ensure gopath/bin
	mkdir -p ${GOPATH}/bin

.PHONY: minimock
minimock: gobin
	curl -sfL https://github.com/gojuno/minimock/releases/download/v2.1.9/minimock_2.1.9_${VERSION}.tar.gz | tar xzf - -C ${GOPATH}/bin/ minimock

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

.PHONY: $(OBSERVER)
$(OBSERVER):
	go build -o $(BIN_DIR)/$(OBSERVER) cmd/observer/*.go

.PHONY: api
api:
	go build -o $(BIN_DIR)/api cmd/api/*.go

.PHONY: xns_stats_count
xns_stats_count:
	go build -o $(BIN_DIR)/xns_stats_count cmd/xns-coin-stats/*.go

$(ARTIFACTS):
	mkdir -p $(ARTIFACTS)

.PHONY: $(CONFIG)
$(CONFIG): $(ARTIFACTS)
	go run ./configuration/gen/gen.go
	mv ./observer.yaml $(ARTIFACTS)/observer.yaml
	mv ./observerapi.yaml $(ARTIFACTS)/observerapi.yaml

ci_test: ## run tests with coverage
	go test -json -v -count 10 -timeout 20m --coverprofile=coverage.txt --covermode=atomic ./... | tee ci_test_with_coverage.json

.PHONY: test
test:
	go test ./... -v

integration:
	go test ./... -tags=integration -v

.PHONY: all
all: env build ## ensure + build + artifacts

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build-docker
build-docker:
	docker build -t insolar/observer -f scripts/docker/Dockerfile .

.PHONY: build-migrate
build-migrate:
	go build -o $(BIN_DIR)/migrate cmd/migrate/*.go

.PHONY: migrate
migrate:
	go run ./cmd/migrate/migrate.go -dir scripts/migrations
