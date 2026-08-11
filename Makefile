GO ?= go
NPM ?= npm
CARGO ?= cargo
BUILD_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || printf unknown)
BUILD_COMMIT_SHORT ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || printf unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_CHANNEL ?= dev-latest
REQUIRE_LOOT_SORTER ?= 0

.PHONY: test build loot-sorter loot-sorter-linux web decky package mvp-audit deck-transfer mvp-release clean

test:
	$(GO) test ./...

build:
	$(GO) build -o bin/dmm-server ./cmd/dmm-server
	$(GO) build -o bin/dmm-nxm-handler ./cmd/dmm-nxm-handler
	$(MAKE) loot-sorter

loot-sorter:
	mkdir -p bin
	cd helpers/loot-sorter && $(CARGO) build --release
	cp helpers/loot-sorter/target/release/dmm-loot-sorter bin/dmm-loot-sorter

loot-sorter-linux:
	mkdir -p bin
	cd helpers/loot-sorter && \
		RUSTC="$$(rustup which rustc 2>/dev/null || command -v rustc)" && \
		if [ "$$(uname -s)" = "Linux" ]; then \
			$(CARGO) build --release --target x86_64-unknown-linux-gnu; \
		else \
			mkdir -p target; \
			printf '%s\n' '#!/bin/sh' 'exec zig cc -target x86_64-linux-gnu "$$@"' > target/dmm-zigcc-linux-amd64; \
			chmod +x target/dmm-zigcc-linux-amd64; \
			CARGO_TARGET_X86_64_UNKNOWN_LINUX_GNU_LINKER="$$PWD/target/dmm-zigcc-linux-amd64" \
				RUSTC="$$RUSTC" \
				$(CARGO) build --release --target x86_64-unknown-linux-gnu; \
		fi
	cp helpers/loot-sorter/target/x86_64-unknown-linux-gnu/release/dmm-loot-sorter bin/dmm-loot-sorter-linux-amd64

web:
	cd web && $(NPM) install && $(NPM) run build

decky:
	cd decky && $(NPM) install && $(NPM) run build
	mkdir -p dist/decky-mod-manager/bin dist/decky-mod-manager/web
	cp bin/dmm-server dist/decky-mod-manager/bin/dmm-server
	cp decky/plugin.json decky/package.json decky/main.py dist/decky-mod-manager/
	cp -R decky/dist dist/decky-mod-manager/dist
	cp -R web/dist dist/decky-mod-manager/web/dist
	chmod +x dist/decky-mod-manager/bin/dmm-server
	printf '%s\n' '{' '  "commit": "$(BUILD_COMMIT)",' '  "short_commit": "$(BUILD_COMMIT_SHORT)",' '  "built_at": "$(BUILD_TIME)",' '  "channel": "$(BUILD_CHANNEL)"' '}' > dist/decky-mod-manager/build-info.json

package:
	GOOS=linux GOARCH=amd64 $(GO) build -o bin/dmm-server-linux-amd64 ./cmd/dmm-server
	GOOS=linux GOARCH=amd64 $(GO) build -o bin/dmm-nxm-handler-linux-amd64 ./cmd/dmm-nxm-handler
	@if [ "$(REQUIRE_LOOT_SORTER)" = "1" ]; then \
		$(MAKE) loot-sorter-linux; \
	elif [ ! -x bin/dmm-loot-sorter-linux-amd64 ]; then \
		echo "==> Skipping Linux LOOT sorter helper; set REQUIRE_LOOT_SORTER=1 in a Linux build environment to require it."; \
	fi
	cd web && $(NPM) install && $(NPM) run build
	cd decky && $(NPM) install && $(NPM) run build
	rm -rf dist/decky-mod-manager dist/decky-mod-manager.tar.gz dist/decky-mod-manager.zip
	mkdir -p dist/decky-mod-manager/bin dist/decky-mod-manager/web dist/decky-mod-manager/dist
	cp bin/dmm-server-linux-amd64 dist/decky-mod-manager/bin/dmm-server
	cp bin/dmm-nxm-handler-linux-amd64 dist/decky-mod-manager/bin/dmm-nxm-handler
	if [ -x bin/dmm-loot-sorter-linux-amd64 ]; then cp bin/dmm-loot-sorter-linux-amd64 dist/decky-mod-manager/bin/dmm-loot-sorter; fi
	cp decky/plugin.json decky/package.json decky/main.py dist/decky-mod-manager/
	cp -R decky/dist/. dist/decky-mod-manager/dist/
	cp -R web/dist dist/decky-mod-manager/web/
	chmod +x dist/decky-mod-manager/bin/dmm-server dist/decky-mod-manager/bin/dmm-nxm-handler
	if [ -f dist/decky-mod-manager/bin/dmm-loot-sorter ]; then chmod +x dist/decky-mod-manager/bin/dmm-loot-sorter; fi
	printf '%s\n' '{' '  "commit": "$(BUILD_COMMIT)",' '  "short_commit": "$(BUILD_COMMIT_SHORT)",' '  "built_at": "$(BUILD_TIME)",' '  "channel": "$(BUILD_CHANNEL)"' '}' > dist/decky-mod-manager/build-info.json
	COPYFILE_DISABLE=1 tar --no-xattrs -C dist -czf dist/decky-mod-manager.tar.gz decky-mod-manager 2>/dev/null || COPYFILE_DISABLE=1 tar -C dist -czf dist/decky-mod-manager.tar.gz decky-mod-manager
	cd dist && COPYFILE_DISABLE=1 zip -qr decky-mod-manager.zip decky-mod-manager

mvp-audit:
	./testing/mvp_audit.sh

deck-transfer: package
	./testing/create_deck_transfer_bundle.sh

mvp-release: mvp-audit
	./testing/create_deck_transfer_bundle.sh

clean:
	rm -rf bin dist web/dist decky/dist
