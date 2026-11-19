.PHONY: build

build:
	go build -ldflags "-X main.version=$$(cat VERSION 2>/dev/null || echo dev) -X main.clientID=$$(jq -r .clientId app_config.json)" -o bin/neat ./cmd
