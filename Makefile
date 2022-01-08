.PHONY: *
.DEFAULT_GOAL:=help

# Linker tags
# https://golang.org/cmd/link/
LD_FLAGS += -s -w

# "buf" is used to manage protocol buffer definitions, either
# locally (on a dev container) or using a builder image.
buf:=buf
ifndef REMOTE_CONTAINERS_SOCKETS
	buf=docker run --platform linux/amd64 --rm -it -v $(shell pwd):/workdir ghcr.io/bryk-io/buf-builder:1.0.0-rc10 buf
endif

# For commands that require a specific package path, default to all local
# subdirectories if no value is provided.
pkg?="..."

help:
	@echo "Commands available"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /' | sort

## bench: Run benchmarks
bench:
	go test -run=XXX -bench=. -benchmem ./...

## build: Build for the current architecture in use, intended for development
build:
	go build -v -ldflags '$(LD_FLAGS)'

## deps: Verify dependencies and remove intermediary products
deps:
	@-rm -rf vendor
	go mod tidy
	go mod verify
	go mod download
	go mod vendor

## docs: Display package documentation on local server
docs:
	@echo "Docs available at: http://localhost:8080/pkg/go.bryk.io/pkg"
	godoc -http=:8080 -goroot=${GOPATH} -play

## lint: Static analysis
lint:
	# Go code
	golangci-lint run -v ./$(pkg)

## scan: Look for known vulnerabilities in the project dependencies
# https://github.com/sonatype-nexus-community/nancy
scan:
	@go list -mod=readonly -f '{{if not .Indirect}}{{.}}{{end}}' -m all | nancy sleuth --skip-update-check

## test: Run all unitary tests
test:
	# Unit tests
	# -count=1 -p=1 (disable cache and parallel execution)
	go test -race -v -failfast -count=1 -p=1 -coverprofile=coverage.report ./$(pkg)
	go tool cover -html coverage.report -o coverage.html

## updates: List available updates for direct dependencies
# https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies
updates:
	@go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -mod=mod -m all 2> /dev/null

## protos: Compile all PB definitions and RPC services
protos:
	# Generate package images and code
	make proto-build pkg=sample/v1

## proto-test: Verify PB definitions on 'pkg'
proto-test:
	# Verify style and consistency
	$(buf) lint --path proto/$(pkg)

	# Verify breaking changes. This fails if no image is already present,
	# use `buf build --o proto/$(pkg)/image.bin --path proto/$(pkg)` to generate it.
	$(buf) breaking --against proto/$(pkg)/image.bin

## proto-build: Build PB definitions on 'pkg'
proto-build:
	# Verify PB definitions
	make proto-test pkg=$(pkg)

	# Build package image
	$(buf) build --output proto/$(pkg)/image.bin --path proto/$(pkg)

	# Generate package code using buf.gen.yaml
	$(buf) generate --output proto --path proto/$(pkg)

	# Remove package comment added by the gateway generator to avoid polluting
	# the package documentation.
	@-sed -i.bak '/\/\*/,/*\//d' proto/$(pkg)/*.pb.gw.go

	# Remove non-required dependencies. "protoc-gen-validate" don't have runtime
	# dependencies but the generated code includes the package by the default =/.
	@-sed -i.bak '/protoc-gen-validate/d' proto/$(pkg)/*.pb.go

	# Remove in-place edit backup files
	@-rm proto/$(pkg)/*.bak

	# Style adjustments (required for consistency)
	gofmt -s -w proto/$(pkg)
	goimports -w proto/$(pkg)
