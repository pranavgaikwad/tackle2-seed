PREPARE = -o bin/prepare github.com/konveyor/tackle2-seed/cmd/prepare
RULESET = -o bin/ruleset github.com/konveyor/tackle2-seed/cmd/ruleset
PKG = ./cmd/... ./pkg/...

cmd: prepare ruleset

prepare: fmt vet
	go build ${PREPARE}

ruleset: fmt vet
	go build ${RULESET}

fmt:
	go fmt ${PKG}

vet:
	go vet ${PKG}

run-prepare: prepare
	bin/prepare resources resources

ruleset-patch: ruleset
	bin/ruleset -p resources
	bin/prepare resources resources
