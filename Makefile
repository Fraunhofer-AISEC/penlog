GO ?= go
version := $(shell git describe --always --dirty)

all: hr pendump penrun

hr:
	$(GO) build $(GOFLAGS) -ldflags="-X main.version=$(version)" -o $@ ./bin/$@/...

pendump:
	$(GO) build $(GOFLAGS) -ldflags="-X main.version=$(version)" -o $@ ./bin/$@/...
	@echo "!! capabilities for pendump needed !!"
	@echo "\"make pendump-caps\" fixes this for you"

pendump-caps: pendump
	sudo setcap cap_dac_override,cap_net_admin,cap_net_raw+eip ./pendump

penrun : bin/penrun/penrun
	cp ./bin/$@/$@ .

man:
	$(MAKE) -C man man

html:
	$(MAKE) -C man html

update:
	$(GO) get -u ./bin/...
	$(GO) mod tidy

clitest:
	$(MAKE) -C tests/cli test

clean:
	$(RM) hr
	$(MAKE) -C man clean

.PHONY: all hr pendump pendump-caps penrun man update clitest clean
