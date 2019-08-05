BIN_DIR = bin
OBSERVER = observer
ARTIFACTS = .artifacts
CONFIG = config

.PHONY: build
build: $(BIN_DIR) $(OBSERVER)

.PHONY: artifacts
artifacts: $(ARTIFACTS) $(CONFIG)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

.PHONY: $(OBSERVER)
$(OBSERVER):
	go build -o $(BIN_DIR)/$(OBSERVER) cmd/observer/*.go

$(ARTIFACTS):
	mkdir -p $(ARTIFACTS)

.PHONY: $(CONFIG)
$(CONFIG):
	go run internal/configuration/gen/gen.go
	mv ./observer.yaml $(ARTIFACTS)/observer.yaml
