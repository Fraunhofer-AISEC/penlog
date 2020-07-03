GO ?= go

all: hr pendump

hr:
	$(GO) build $(GOFLAGS) -o $@ ./bin/$@/...

pendump:
	$(GO) build $(GOFLAGS) -o $@ ./bin/$@/...

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

.PHONY: all hr pendump man update clitest clean
