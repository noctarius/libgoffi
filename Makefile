.PHONY: test precheck clean init

REQ_VERSION_GO := "1.12"

GO ?= $(shell echo `command -v go`)
CMAKE ?= $(shell echo `command -v cmake`)
MAKE ?= $(shell echo `command -v make`)
FILE ?= $(shell echo `command -v file`)

GOARM ?= 7

.init: .precheck
	@echo -n "Preparing buildsystem... "
	$(eval CROSS_ARCH ?= $(shell GO111MODULE=off $(GO) run build/build.go -a))
	$(eval CROSS_OS ?= $(shell GO111MODULE=off $(GO) run build/build.go -o))
	@echo "done."

.info: .init
	@echo "Building for ARCH=$(CROSS_ARCH) and OS=$(CROSS_OS)"

clean:
	@echo -n "Cleaning project... "
	@rm -rf target
	@echo "done."

.precheck:
	@echo -n "Testing for required build tools... "
	@command -v go > /dev/null 2>&1 || \
		{ echo >&2 "Go compiler >=1.12 needs to be available in the path for compilation"; exit 1; }

	@command -v cmake > /dev/null 2>&1 || \
		{ echo >&2 "CMAKE needs to be available in the path for compilation"; exit 1; }

	@command -v make > /dev/null 2>&1 || \
		{ echo >&2 "MAKE needs to be available in the path for compilation"; exit 1; }

	$(eval version="$(shell $(GO) version | awk '{print $$3}' | sed -E 's/go([0-9]*\.[0-9]*\.[0-9]*)/\1/')")
	$(eval higest="$(shell echo $(REQ_VERSION_GO) $(version) | sed -E 's/ /\n/' | sort -V | head -n 1)")
	@test "$(higest)" = "$(REQ_VERSION_GO)" || \
		{ echo ""; echo >&2 "Go compiler >=1.12 needs to be available in the path for compilation, only $(higest) found"; exit 1; }

	@echo "done."

test: .info
	@mkdir -p target
	@cd target
	@cd target && $(CMAKE) ../tests
	@cd target && $(MAKE)
	@$(FILE) target/libgoffitests.so
	@GOARCH=$(CROSS_ARCH) GOOS=$(CROSS_OS) GOARM=$(GOARM) $(GO) test -o target/tests -cover -coverprofile=target/c.out
	@GOARCH=$(CROSS_ARCH) GOOS=$(CROSS_OS) GOARM=$(GOARM) $(GO) tool cover -html=target/c.out -o target/coverage.html
