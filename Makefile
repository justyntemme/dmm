GO ?= go
NPM ?= npm

.PHONY: test build web decky package mvp-audit deck-transfer mvp-release clean

test:
	$(GO) test ./...

build:
	$(GO) build -o bin/dmm-server ./cmd/dmm-server
	$(GO) build -o bin/dmm-nxm-handler ./cmd/dmm-nxm-handler

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

package:
	GOOS=linux GOARCH=amd64 $(GO) build -o bin/dmm-server-linux-amd64 ./cmd/dmm-server
	GOOS=linux GOARCH=amd64 $(GO) build -o bin/dmm-nxm-handler-linux-amd64 ./cmd/dmm-nxm-handler
	cd web && $(NPM) install && $(NPM) run build
	cd decky && $(NPM) install && $(NPM) run build
	rm -rf dist/decky-mod-manager dist/decky-mod-manager.tar.gz dist/decky-mod-manager.zip
	mkdir -p dist/decky-mod-manager/bin dist/decky-mod-manager/web dist/decky-mod-manager/dist
	cp bin/dmm-server-linux-amd64 dist/decky-mod-manager/bin/dmm-server
	cp bin/dmm-nxm-handler-linux-amd64 dist/decky-mod-manager/bin/dmm-nxm-handler
	cp decky/plugin.json decky/package.json decky/main.py dist/decky-mod-manager/
	cp -R decky/dist/. dist/decky-mod-manager/dist/
	cp -R web/dist dist/decky-mod-manager/web/
	chmod +x dist/decky-mod-manager/bin/dmm-server dist/decky-mod-manager/bin/dmm-nxm-handler
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
