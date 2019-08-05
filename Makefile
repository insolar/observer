BIN_DIR ?= bin
OBSERVER = observer
GOBUILD ?= go build

.PHONY: build
build: $(BIN_DIR) $(OBSERVER)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

.PHONY: $(OBSERVER)
$(OBSERVER):
	$(GOBUILD) -o $(BIN_DIR)/$(OBSERVER) cmd/observer/*.go
