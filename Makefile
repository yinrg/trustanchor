APP_NAME   := trustanchor
APP_DISPLAY_NAME := Trust Anchor
APP_ID     := io.github.yinrg.trustanchor
MODULE     := github.com/yinrg/trustanchor
BIN_DIR    := build
PREFIX     ?= $(HOME)/.local
ICON_SIZES := 16 24 32 48 64 128 256 512
WINDOWS_CC := x86_64-w64-mingw32-gcc
VERSION    ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo 0.0.0-dev)
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION_NUM := $(shell echo $(VERSION) | sed -E 's/^v//; s/[^0-9.].*$$//; s/^$$/0.0.0/')

LDFLAGS := -s -w \
	-X '$(MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/version.Date=$(BUILD_DATE)'

ifeq ($(OS),Windows_NT)
BINARY := $(BIN_DIR)/$(APP_NAME).exe
else
BINARY := $(BIN_DIR)/$(APP_NAME)
endif

.PHONY: all run build package test vet fmt tidy icon install uninstall cross cross-windows windows-res package-windows clean

all: build

run:
	go run -ldflags "$(LDFLAGS)" .

build:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

package: build
	go run fyne.io/tools/cmd/fyne@v1.7.4 package --executable $(BINARY) --release
	@for f in *.tar.xz *.deb *.app *.exe; do if [ -e "$$f" ]; then mv "$$f" $(BIN_DIR)/; fi; done

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

icon:
	go run ./tools/genicon

install: build
	install -Dm755 $(BINARY) $(DESTDIR)$(PREFIX)/bin/$(APP_NAME)
	@for s in $(ICON_SIZES); do \
		install -Dm644 assets/icons/$${s}x$${s}.png \
			$(DESTDIR)$(PREFIX)/share/icons/hicolor/$${s}x$${s}/apps/$(APP_ID).png; \
	done
	@mkdir -p $(DESTDIR)$(PREFIX)/share/applications
	sed 's|^Exec=.*|Exec=$(PREFIX)/bin/$(APP_NAME)|' packaging/linux/$(APP_ID).desktop \
		> $(DESTDIR)$(PREFIX)/share/applications/$(APP_ID).desktop
	-gtk-update-icon-cache -f -t $(DESTDIR)$(PREFIX)/share/icons/hicolor >/dev/null 2>&1 || true
	-update-desktop-database $(DESTDIR)$(PREFIX)/share/applications >/dev/null 2>&1 || true

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(APP_NAME)
	rm -f $(DESTDIR)$(PREFIX)/share/applications/$(APP_ID).desktop
	@for s in $(ICON_SIZES); do \
		rm -f $(DESTDIR)$(PREFIX)/share/icons/hicolor/$${s}x$${s}/apps/$(APP_ID).png; \
	done

cross:
	fyne-cross windows -arch amd64 -icon assets/icon.png
	fyne-cross linux -arch amd64 -icon assets/icon.png

windows-res:
	go run github.com/tc-hib/go-winres@v0.3.3 simply \
		--arch amd64 --out rsrc \
		--icon assets/icon.png --manifest gui \
		--product-version $(VERSION_NUM) --file-version $(VERSION_NUM) \
		--product-name "$(APP_DISPLAY_NAME)" --file-description "$(APP_DISPLAY_NAME)" \
		--original-filename $(APP_NAME).exe

cross-windows: icon windows-res
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=$(WINDOWS_CC) \
		go build -trimpath -ldflags "-H windowsgui $(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME).exe .

package-windows: cross-windows
	@echo "Windows 可执行文件: $(BIN_DIR)/$(APP_NAME).exe"

clean:
	rm -rf $(BIN_DIR) fyne-cross
