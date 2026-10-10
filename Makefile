VERSION ?= 0.1.0
PREFIX ?= /usr/local
BIN_DIR ?= $(PREFIX)/bin
MAN_DIR ?= $(PREFIX)/share/man/man1
BASH_COMP_DIR ?= $(PREFIX)/share/bash-completion/completions
ZSH_COMP_DIR ?= $(PREFIX)/share/zsh/site-functions
FISH_COMP_DIR ?= $(PREFIX)/share/fish/vendor_completions.d

.PHONY: all build test deb install uninstall clean

all: build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o bin/rot13 ./cmd/rot13

test:
	go vet ./...
	go test -v -race ./...

deb:
	./scripts/build_deb.sh $(VERSION) amd64

install: build
	install -d $(DESTDIR)$(BIN_DIR)
	install -m 755 bin/rot13 $(DESTDIR)$(BIN_DIR)/rot13
	install -d $(DESTDIR)$(MAN_DIR)
	install -m 644 man/rot13.1 $(DESTDIR)$(MAN_DIR)/rot13.1
	install -d $(DESTDIR)$(BASH_COMP_DIR)
	install -m 644 completions/rot13.bash $(DESTDIR)$(BASH_COMP_DIR)/rot13
	install -d $(DESTDIR)$(ZSH_COMP_DIR)
	install -m 644 completions/rot13.zsh $(DESTDIR)$(ZSH_COMP_DIR)/_rot13
	install -d $(DESTDIR)$(FISH_COMP_DIR)
	install -m 644 completions/rot13.fish $(DESTDIR)$(FISH_COMP_DIR)/rot13.fish

uninstall:
	rm -f $(DESTDIR)$(BIN_DIR)/rot13
	rm -f $(DESTDIR)$(MAN_DIR)/rot13.1
	rm -f $(DESTDIR)$(BASH_COMP_DIR)/rot13
	rm -f $(DESTDIR)$(ZSH_COMP_DIR)/_rot13
	rm -f $(DESTDIR)$(FISH_COMP_DIR)/rot13.fish

clean:
	rm -rf bin dist rot13 rot13.exe
