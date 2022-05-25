GO ?= go
version := $(shell git describe --always --dirty)

all: hr

.PHONY: hr
hr:
	$(GO) build $(GOFLAGS) -ldflags="-X main.version=$(version)" -o $@ ./bin/$@/...

.PHONY: update
update:
	$(GO) get -u ./bin/...
	$(GO) mod tidy

.PHONY: clitest
clitest:
	$(MAKE) -C tests/cli test

.PHONY: clean
clean:
	$(RM) hr
