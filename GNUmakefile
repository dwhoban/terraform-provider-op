default: fmt lint install generate

build:
	go build -v ./...

# Build into $(GOPATH)/bin (not GOBIN, which may sit inside a versioned Go
# toolchain dir) so a ~/.terraformrc dev_overrides entry stays stable.
install: build
	go build -v -o $(shell go env GOPATH)/bin/terraform-provider-op .

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

.PHONY: fmt lint test testacc build install generate
