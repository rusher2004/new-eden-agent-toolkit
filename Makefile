.PHONY: urlscheme-macos-bundle

BUNDLE := urlscheme/macos/build/NeatURLHandler.app

urlscheme-macos-bundle:
	mkdir -p $(BUNDLE)/Contents/MacOS
	swiftc urlscheme/macos/main.swift -o $(BUNDLE)/Contents/MacOS/NeatURLHandler
	cp urlscheme/macos/Info.plist $(BUNDLE)/Contents/Info.plist
