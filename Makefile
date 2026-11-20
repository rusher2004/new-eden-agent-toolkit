.PHONY: build install

PREFIX ?= $(HOME)/.local

build:
	go build -ldflags "-X main.version=$$(cat VERSION 2>/dev/null || echo dev) -X main.clientID=$$(jq -r '.esiClientId // .clientId' app_config.json) -X main.aptabaseAppKey=$$(jq -r '.aptabaseAppKey // ""' app_config.json) -X main.analyticsDebug=true" -o bin/neat ./cmd

# Remove the old binary before copying rather than overwriting it in place:
# macOS caches a binary's code signature per file, so copying a new build over
# an existing one makes the kernel SIGKILL it on every launch (exit 137).
install: build
	mkdir -p $(PREFIX)/bin
	rm -f $(PREFIX)/bin/neat
	cp bin/neat $(PREFIX)/bin/neat
