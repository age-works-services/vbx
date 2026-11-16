.DEFAULT_GOAL := all
.PHONY: all help build install test check cross-check

PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
DESTDIR ?=

# Recursive invocations keep tests and installation ordered, including with -j.
all: build
	$(MAKE) test
	$(MAKE) install

help:
	@printf '%s\n' \
	  'Targets:' \
	  '  all          Build, test, then install (default)' \
	  '  help         Show available targets and installation options' \
	  '  build        Build the standalone executable in bin/vbx' \
	  '  install      Build and install vbx to BINDIR' \
	  '  test         Run Go tests and offline shell suites' \
	  '  check        Run go vet and race-enabled Go tests' \
	  '  cross-check  Check Darwin ARM64 compilation' \
	  '' \
	  'Installation options:' \
	  '  PREFIX       Installation prefix (default: $$HOME/.local)' \
	  '  BINDIR       Executable directory (default: PREFIX/bin)' \
	  '  DESTDIR      Optional staging root prepended to BINDIR'

build:
	CGO_ENABLED=0 go build -trimpath -o bin/vbx ./cmd/vbx

install: build
	install -d "$(DESTDIR)$(BINDIR)"
	install -m 755 bin/vbx "$(DESTDIR)$(BINDIR)/vbx"

test:
	go test ./...
	@for suite in test/test-*.sh; do bash "$$suite" || exit; done
	@for suite in modules/*/test/test-*.sh; do [ -f "$$suite" ] || continue; bash "$$suite" || exit; done

# Race validation requires cgo and a host C compiler, independently of release builds.
check:
	go vet ./...
	CGO_ENABLED=1 go test -race ./...

# Compilation only. VM operation on macOS remains unsupported.
cross-check:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o bin/vbx-darwin-arm64 ./cmd/vbx
