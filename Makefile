GO ?= go
version := $(shell git describe --always --dirty)

all: hr

.PHONY: hr
hr:
	$(GO) build $(GOFLAGS) -ldflags="-X main.version=$(version)" -o $@ ./bin/$@/...

man:
	$(MAKE) -C man man

html:
	$(MAKE) -C man html

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
	$(MAKE) -C man clean
