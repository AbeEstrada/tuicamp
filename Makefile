PREFIX ?= /usr/local
PKG_NAME := $(shell basename "$(shell go list)")
BUILD_FLAGS := -ldflags="-s -w" -trimpath

.PHONY: build install uninstall clean

build:
	@echo "Building $(PKG_NAME) with release optimizations..."
	go build $(BUILD_FLAGS) -o $(PKG_NAME) .

install: build
	@echo "Installing to $(DESTDIR)$(PREFIX)/bin"
	@mkdir -p $(DESTDIR)$(PREFIX)/bin
	@mv $(PKG_NAME) $(DESTDIR)$(PREFIX)/bin/
	@echo "Installation complete. $(PKG_NAME) is now available in $(DESTDIR)$(PREFIX)/bin"

uninstall:
	@echo "Removing $(DESTDIR)$(PREFIX)/bin/$(PKG_NAME)"
	@rm -f $(DESTDIR)$(PREFIX)/bin/$(PKG_NAME)
	@echo "Uninstallation complete"

clean:
	@rm -f $(PKG_NAME)
