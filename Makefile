.PHONY: *
.DEFAULT_GOAL:=help

# Linker tags
# https://golang.org/cmd/link/
LD_FLAGS += -s -w

# For commands that require a specific package path, default to all local
# subdirectories if no value is provided.
pkg?="..."

help:
	@echo "Commands available"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /' | sort

## bench: Run benchmarks
bench:
	go test -run=XXX -bench=. -benchmem ./$(pkg)

## build: Build for the current architecture in use, intended for development
build:
	go build -v -ldflags '$(LD_FLAGS)'

## codeql: Run a CodeQL scan operation locally
# https://codeql.github.com/docs/
codeql:
	codeql database create --overwrite .codeql/db --language go
	codeql database analyze .codeql/db --format=sarif-latest --output=.codeql/issues.sarif

## deps: Verify dependencies and remove intermediary products
deps:
	go mod tidy
	go clean

## docs: Display package documentation on local server
docs:
	@echo "Docs available at: http://localhost:8080/pkg/go.bryk.io/pkg/"
	godoc -http=:8080 -goroot=${GOPATH} -play

## fuzz: Run fuzz tests for 30 seconds
fuzz:
	go test -fuzz=. -fuzztime 30s -v ./$(pkg)

## lint: Static analysis
lint:
	# Go code
	golangci-lint run -v ./$(pkg)

## protos: Compile all protobuf definitions and RPC services
protos:
	# Generate package images and code
	make proto-build pkg=sample/v1

## scan-ci: Look for vulnerabilities in CI Workflows
# https://docs.zizmor.sh/usage/
scan-ci:
	actionlint
	zizmor --gh-token `gh auth token` .github/workflows

## scan-deps: Scan code and dependencies for known vulnerabilities
# https://appsec.guide/docs/static-analysis/semgrep/
# https://go.googlesource.com/vuln
scan-deps:
	govulncheck -mode source -scan package ./...
	semgrep --config "p/trailofbits"

## scan-secrets: Scan project code for accidentally leaked secrets
# https://gitleaks.io
# gitleaks dir --no-banner -f json -r - | jq -r '.[].Fingerprint' > .gitleaksignore
scan-secrets:
	gitleaks git -v

## test: Run all unitary tests
test:
	# Unit tests
	# -count=1 -p=1 (disable cache and parallel execution)
	go test -race -v -coverprofile=coverage.report ./$(pkg)
	go tool cover -html coverage.report -o coverage.html
	# https://github.com/nikolaydubina/go-cover-treemap
	# go-cover-treemap -coverprofile coverage.report > coverage.svg

## updates: List available updates for direct dependencies
# https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies
updates:
	@GOWORK=off go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: [{{.Version}} -> {{.Update.Version}}]{{end}}' -mod=mod -m all 2> /dev/null

proto-test:
	# Verify style and consistency
	buf format --exit-code -d
	buf lint

	# Verify breaking changes. This fails if no image is already present,
	# use `buf build -o proto/sample/v1/image.bin` to generate it.
	buf breaking --against proto/sample/v1/image.bin

proto-build:
	# Verify PB definitions
	make proto-test

	# Build package image
	buf build -o proto/sample/v1/image.bin

	# Generate package code using buf.gen.yaml
	buf generate -o proto

	# Add compiler version to generated files
	@-sed -i.bak 's/(unknown)/buf-v$(shell buf --version)/g' proto/$(pkg)/*.pb.go

	# Remove package comment added by the gateway generator to avoid polluting
	# the package documentation.
	@-sed -i.bak '/\/\*/,/*\//d' proto/$(pkg)/*.pb.gw.go

	# "protoc-gen-openapiv2" don't have runtime dependencies but the generated
	# code includes the package by the default =/
	@-sed -i.bak '/protoc-gen-openapiv2/d' proto/$(pkg)/*.pb.go

	# Remove in-place edit backup files
	@-rm proto/$(pkg)/*.bak

	# Style adjustments (required for consistency)
	gofmt -s -w proto/$(pkg)
	goimports -w proto/$(pkg)
