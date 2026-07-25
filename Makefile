.PHONY: urlscheme-macos-bundle build

BUNDLE := urlscheme/macos/build/NeatURLHandler.app

urlscheme-macos-bundle:
	mkdir -p $(BUNDLE)/Contents/MacOS
	swiftc urlscheme/macos/main.swift -o $(BUNDLE)/Contents/MacOS/NeatURLHandler
	cp urlscheme/macos/Info.plist $(BUNDLE)/Contents/Info.plist

build:
	go build -ldflags "-X main.version=dev -X main.clientID=$$(jq -r .clientId app_config.json)" -o bin/neat ./cmd
