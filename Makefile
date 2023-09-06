BUILD = -o bin/prepare github.com/konveyor/tackle2-seed/cmd
PKG = ./cmd/... ./pkg/...

cmd: fmt vet
	go build ${BUILD}

fmt:
	go fmt ${PKG}

vet:
	go vet ${PKG}

prepare: cmd
	bin/prepare resources resources
