.PHONY: *
.DEFAULT_GOAL:=help

# Linker tags
# https://golang.org/cmd/link/
LD_FLAGS += -s -w

# "buf" is used to manage protocol buffer definitions, either
# locally (on a dev container) or using a builder image.
buf:=buf
ifndef REMOTE_CONTAINERS_SOCKETS
	buf=docker run --platform linux/amd64 --rm -it -v $(shell pwd):/workdir ghcr.io/bryk-io/buf-builder:1.9.0 buf
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

## codeql: Run a CodeQL scan operation locally
# https://codeql.github.com/docs/
codeql:
	@-rm -rf codeql-results.csv codeql-db
	codeql database create --overwrite --language go codeql-db
	codeql database analyze codeql-db --format csv --output codeql-results.csv

## deps: Verify dependencies and remove intermediary products
deps:
	go mod tidy
	go clean

## docs: Display package documentation on local server
docs:
	@echo "Docs available at: http://localhost:8080/pkg/go.bryk.io/pkg"
	godoc -http=:8080 -goroot=${GOPATH} -play

## lint: Static analysis
lint:
	# Go code
	golangci-lint run -v ./$(pkg)

## scan-deps: Look for known vulnerabilities in the project dependencies
# https://github.com/sonatype-nexus-community/nancy
scan-deps:
	@go list -json -deps ./... | nancy sleuth --skip-update-check

## scan-secrets: Scan project code for accidentally leaked secrets
# https://github.com/trufflesecurity/trufflehog
scan-secrets:
	@docker run -it --rm --platform linux/arm64 \
	-v "$PWD:/repo" \
	trufflesecurity/trufflehog:latest \
	filesystem --directory /repo --only-verified

## test: Run all unitary tests
test:
	# Unit tests
	# -count=1 -p=1 (disable cache and parallel execution)
	go test -race -v -failfast -count=1 -p=1 -coverprofile=coverage.report ./$(pkg)
	go tool cover -html coverage.report -o coverage.html

## updates: List available updates for direct dependencies
# https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies
updates:
	@GOWORK=off go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}} [{{.Version}} -> {{.Update.Version}}]{{end}}' -m all 2> /dev/null

## protos: Compile all protobuf definitions and RPC services
protos:
	# Generate package images and code
	make proto-build pkg=sample/v1

proto-test:
	# Verify style and consistency
	$(buf) lint --path proto/$(pkg)

	# Verify breaking changes. This fails if no image is already present,
	# use `buf build --o proto/$(pkg)/image.bin --path proto/$(pkg)` to generate it.
	$(buf) breaking --against proto/$(pkg)/image.bin

proto-build:
	# Verify PB definitions
	make proto-test pkg=$(pkg)

	# Build package image
	$(buf) build --output proto/$(pkg)/image.bin --path proto/$(pkg)

	# Generate package code using buf.gen.yaml
	$(buf) generate --output proto --path proto/$(pkg)

	# Add compiler version to generated files
	@-sed -i.bak 's/(unknown)/buf-v$(shell buf --version)/g' proto/$(pkg)/*.pb.go

	# Remove package comment added by the gateway generator to avoid polluting
	# the package documentation.
	@-sed -i.bak '/\/\*/,/*\//d' proto/$(pkg)/*.pb.gw.go

	# "protoc-gen-validate" don't have runtime dependencies but the generated
	# code includes the package by the default =/
	@-sed -i.bak '/protoc-gen-validate/d' proto/$(pkg)/*.pb.go

	# "protoc-gen-openapiv2" don't have runtime dependencies but the generated
	# code includes the package by the default =/
	@-sed -i.bak '/protoc-gen-openapiv2/d' proto/$(pkg)/*.pb.go

	# Remove in-place edit backup files
	@-rm proto/$(pkg)/*.bak

	# Style adjustments (required for consistency)
	gofmt -s -w proto/$(pkg)
	goimports -w proto/$(pkg)
