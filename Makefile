PREFIX ?= $(HOME)/.local
BIN_DIR := bin
BINARY := $(BIN_DIR)/azprofile
GO ?= go
LDFLAGS ?= -s -w

.PHONY: all build install uninstall clean list

all: build

build: $(BINARY)

$(BINARY):
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/azprofile

install: build
	@mkdir -p $(PREFIX)/bin
	@ln -sf $(CURDIR)/$(BINARY) $(PREFIX)/bin/azprofile
	@echo "  linked $(PREFIX)/bin/azprofile"

uninstall:
	@rm -f $(PREFIX)/bin/azprofile
	@echo "  removed $(PREFIX)/bin/azprofile"

clean:
	@rm -rf $(BIN_DIR) dist

list:
	@echo "Available tools:"
	@echo "  azprofile (Go binary)"
