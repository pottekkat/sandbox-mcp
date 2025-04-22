.PHONY: build clean deps images

# Install dependencies
deps:
	go mod tidy 

# Build the application
build:
	mkdir -p dist
	go build -ldflags="-X 'github.com/pottekkat/sandbox-mcp/internal/version.Version=$$(git describe --tags)' -X 'github.com/pottekkat/sandbox-mcp/internal/version.CommitSHA=$$(git rev-parse --short HEAD)'" -o dist/sandbox-mcp ./cmd/sandbox-mcp/main.go

# Install the application
install:
	go install -ldflags="-X 'github.com/pottekkat/sandbox-mcp/internal/version.Version=$$(git describe --tags)' -X 'github.com/pottekkat/sandbox-mcp/internal/version.CommitSHA=$$(git rev-parse --short HEAD)'" ./cmd/sandbox-mcp

# Clean build artifacts
clean:
	rm -rf dist/sandbox-mcp

# Build sandbox images
images:
	docker build --file sandboxes/shell/Dockerfile --tag sandbox-mcp/shell:latest sandboxes/shell/
	docker build --file sandboxes/go/Dockerfile --tag sandbox-mcp/go:latest sandboxes/go/
	docker build --file sandboxes/python/Dockerfile --tag sandbox-mcp/python:latest sandboxes/python/
	docker build --file sandboxes/javascript/Dockerfile --tag sandbox-mcp/javascript:latest sandboxes/javascript/
	docker build --file sandboxes/network-tools/Dockerfile --tag sandbox-mcp/network-tools:latest sandboxes/network-tools/
