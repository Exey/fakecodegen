BINARY := fakecodegen
INSTALL_DIR := $(shell go env GOPATH)/bin

.PHONY: build install clean run-example

build:
	go build -o $(BINARY) .

install: build
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "installed → $(INSTALL_DIR)/$(BINARY)"

clean:
	rm -f $(BINARY)

# Quick demo: reconstruct a fake repo from prompt-Go.md
run-example: build
	./$(BINARY) -from-prompt prompt-Go.md -folder ./fake-repo
