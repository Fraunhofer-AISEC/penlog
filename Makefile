GO ?= go

all: hr pendump

hr:
	$(GO) build $(GOFLAGS) -o $@ ./bin/$@/...

pendump:
	$(GO) build $(GOFLAGS) -o $@ ./bin/$@/...

pendump-caps: pendump
	sudo setcap cap_dac_override,cap_net_admin,cap_net_raw+eip ./pendump

man:
	$(MAKE) -C man

update:
	$(GO) get -u ./bin/...
	$(GO) mod tidy

clitest:
	$(MAKE) -C tests/cli test

clean:
	$(RM) hr
	$(MAKE) -C man clean

.PHONY: all hr pendump pendump-caps man update clitest clean
