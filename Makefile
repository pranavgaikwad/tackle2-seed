GOPATH ?= $(HOME)/go
GOBIN ?= $(GOPATH)/bin
GOIMPORTS = $(GOBIN)/goimports

PREPARE = -o bin/prepare github.com/konveyor/tackle2-seed/cmd/prepare
RULESET = -o bin/ruleset github.com/konveyor/tackle2-seed/cmd/ruleset

PKG = ./cmd/... \
      ./pkg/...

PKGDIR = $(subst /...,,$(PKG))

RULESET_ARGS ?=

cmd: prepare ruleset

prepare: fmt vet
	go build $(PREPARE)

ruleset: fmt vet
	go build $(RULESET)

fmt: $(GOIMPORTS)
	$(GOIMPORTS) -w $(PKGDIR)

vet:
	go vet $(PKG)

run-prepare: prepare
	bin/prepare

ruleset-patch: cmd
	bin/prepare
	bin/ruleset $(RULESET_ARGS)
	bin/prepare

# Ensure goimports installed.
$(GOIMPORTS):
	go install golang.org/x/tools/cmd/goimports@latest

