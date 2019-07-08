BIN_DIR ?= bin
ARTIFACTS_DIR ?= .artifacts
INSOLARD = insolard

ALL_PACKAGES = ./...
MOCKS_PACKAGE = github.com/insolar/observer/testutils
GOBUILD ?= go build
FUNCTEST_COUNT ?= 1
TESTED_PACKAGES ?= $(shell go list ${ALL_PACKAGES} | grep -v "${MOCKS_PACKAGE}")
COVERPROFILE ?= coverage.txt
TEST_ARGS ?= -timeout 1200s
BUILD_TAGS ?=

CI_GOMAXPROCS ?= 8
CI_TEST_ARGS ?= -p 4

BUILD_NUMBER := $(TRAVIS_BUILD_NUMBER)
BUILD_DATE = $(shell date "+%Y-%m-%d")
BUILD_TIME = $(shell date "+%H:%M:%S")
BUILD_HASH = $(shell git rev-parse --short HEAD)
BUILD_VERSION ?= $(shell git describe --abbrev=0 --tags)

GOPATH ?= `go env GOPATH`
LDFLAGS += -X github.com/insolar/insolar/version.Version=${BUILD_VERSION}
LDFLAGS += -X github.com/insolar/insolar/version.BuildNumber=${BUILD_NUMBER}
LDFLAGS += -X github.com/insolar/insolar/version.BuildDate=${BUILD_DATE}
LDFLAGS += -X github.com/insolar/insolar/version.BuildTime=${BUILD_TIME}
LDFLAGS += -X github.com/insolar/insolar/version.GitHash=${BUILD_HASH}


.PHONY: all
all: clean install-deps pre-build build

.PHONY: lint
lint: ci-lint

.PHONY: ci-lint
ci-lint:
	golangci-lint run

.PHONY: metalint
metalint:
	gometalinter --vendor $(ALL_PACKAGES)

.PHONY: clean
clean:
	go clean $(ALL_PACKAGES)
	rm -f $(COVERPROFILE)
	rm -rf $(BIN_DIR)


.PHONY: install-godep
install-godep:
	$GOPATH/src/github.com/insolar/insolar/scripts/build/fetchdeps github.com/golang/dep/cmd/dep v0.5.3

.PHONY: install-build-tools
install-build-tools:
	go clean -modcache
	$GOPATH/src/github.com/insolar/insolar/scripts/build/fetchdeps golang.org/x/tools/cmd/stringer 63e6ed9258fa6cbc90aab9b1eef3e0866e89b874
	$GOPATH/src/github.com/insolar/insolar/scripts/build/fetchdeps github.com/gojuno/minimock/cmd/minimock 890c67cef23dd06d694294d4f7b1026ed7bac8e6
	$GOPATH/src/github.com/insolar/insolar/scripts/build/fetchdeps github.com/gogo/protobuf/protoc-gen-gogoslick v1.2.1

.PHONY: install-deps
install-deps: install-godep install-build-tools

.PHONY: pre-build
pre-build: ensure generate

.PHONY: generate
generate:
	GOPATH=`go env GOPATH` go generate -x $(ALL_PACKAGES)

.PHONY: ensure
ensure:
	dep ensure

.PHONY: build
build: $(BIN_DIR) $(INSOLARD)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

.PHONY: $(INSOLARD)
$(INSOLARD):
	$(GOBUILD) -o $(BIN_DIR)/$(INSOLARD) ${BUILD_TAGS} -ldflags "${LDFLAGS}" cmd/insolard/*.go

.PHONY: test_unit
test_unit:
	CGO_ENABLED=1 go test $(TEST_ARGS) $(ALL_PACKAGES)

.PHONY: functest
functest:
	CGO_ENABLED=1 go test -test.v $(TEST_ARGS) -tags functest ./functest -count=$(FUNCTEST_COUNT)

.PNONY: functest_race
functest_race:
	make clean
	GOBUILD='go build -race' make build
	FUNCTEST_COUNT=10 make functest

.PHONY: test_func
test_func: functest

.PHONY: test_slow
test_slow:
	CGO_ENABLED=1 go test $(TEST_ARGS) -tags slowtest ./logicrunner/... ./server/internal/...

.PHONY: test
test: test_unit

.PHONY: test_all
test_all: test_unit test_func test_slow

.PHONY: test_with_coverage
test_with_coverage: $(ARTIFACTS_DIR)
	CGO_ENABLED=1 go test $(TEST_ARGS) --coverprofile=$(ARTIFACTS_DIR)/cover.all --covermode=atomic $(TESTED_PACKAGES)
	@cat $(ARTIFACTS_DIR)/cover.all | ./scripts/dev/cover-filter.sh > $(COVERPROFILE)

.PHONY: test_with_coverage_fast
test_with_coverage_fast:
	CGO_ENABLED=1 go test $(TEST_ARGS) -count 1 --coverprofile=$(COVERPROFILE) --covermode=atomic $(ALL_PACKAGES)

$(ARTIFACTS_DIR):
	mkdir -p $(ARTIFACTS_DIR)

.PHONY: ci_test_with_coverage
ci_test_with_coverage:
	GOMAXPROCS=$(CI_GOMAXPROCS) CGO_ENABLED=1 \
		go test $(CI_TEST_ARGS) $(TEST_ARGS) -json -v -count 1 --coverprofile=$(COVERPROFILE) --covermode=atomic -tags slowtest $(ALL_PACKAGES)

.PHONY: ci_test_unit
ci_test_unit:
	GOMAXPROCS=$(CI_GOMAXPROCS) CGO_ENABLED=1 \
		go test $(CI_TEST_ARGS) $(TEST_ARGS) -json -v $(ALL_PACKAGES) -race -count 10 | tee ci_test_unit.json

.PHONY: ci_test_slow
ci_test_slow:
	GOMAXPROCS=$(CI_GOMAXPROCS) CGO_ENABLED=1 \
		go test $(CI_TEST_ARGS) $(TEST_ARGS) -json -v -tags slowtest ./logicrunner/... ./server/internal/... -count 1 | tee -a ci_test_unit.json

.PHONY: docker-insolard
docker-insolard:
	docker build --target insolard --tag insolar/insolard -f ./docker/Dockerfile .

.PHONY: docker
docker: docker-insolard

generate-protobuf:
	protoc -I./vendor -I./ --gogoslick_out=./ network/node/internal/node/node.proto
	protoc -I./vendor -I./ --gogoslick_out=./ insolar/record/record.proto
	protoc -I./vendor -I./ --gogoslick_out=./ --proto_path=${GOPATH}/src insolar/payload/payload.proto
	protoc -I./vendor -I./ --gogoslick_out=./ ledger/object/lifeline.proto
	protoc -I./vendor -I./ --gogoslick_out=./ ledger/object/filamentindex.proto
	protoc -I./vendor -I./ --gogoslick_out=./ insolar/pulse/pulse.proto
	protoc -I./vendor -I./ --gogoslick_out=./ --proto_path=${GOPATH}/src network/hostnetwork/packet/packet.proto
