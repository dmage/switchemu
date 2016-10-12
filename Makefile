GODIR := .go
GO_TGZ := go1.7.$(shell uname -s | tr '[A-Z]' '[a-z]')-$(shell uname -m | sed -e 's/x86_64/amd64/').tar.gz

all:
	false

$(GODIR)/$(GO_TGZ):
	mkdir -p $(GODIR)
	wget "https://storage.googleapis.com/golang/$(GO_TGZ)" -O $@

$(GODIR)/go: $(GODIR)/$(GO_TGZ)
	cd $(GODIR) && tar xf "$(GO_TGZ)"

$(GODIR)/gopath:
	mkdir -p $(GODIR)/gopath/src/github.com/dmage
	ln -s ../../../../.. $(GODIR)/gopath/src/github.com/dmage/switchemu

$(GODIR): $(GODIR)/go $(GODIR)/gopath
	@echo "export GOROOT=$(PWD)/$(GODIR)/go"
	@echo "export GOPATH=$(PWD)/$(GODIR)/gopath"
	@echo "export PATH=\"$(PWD)/$(GODIR)/go/bin:$(PWD)/$(GODIR)/gopath/bin:\$$PATH\""

clean:
	-rm -r ./output/*

.PHONY: $(GODIR) clean
