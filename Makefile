.PHONY: build run test clean format lint help

# Build the application
build:
	go build -o brainmcp main.go constants.go embedder.go handlers.go cli.go

# Run in interactive mode
test:
	export GEMINI_API_KEY="your-api-key" && ./brainmcp -t

# Run as MCP server
run: build
	./brainmcp

# Clean build artifacts
clean:
	rm -f brainmcp

# Format code
format:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

# Display help
help:
	@echo "BrainMCP Build Targets:"
	@echo "  build  - Compile the application"
	@echo "  test   - Run interactive CLI test mode"
	@echo "  run    - Build and run as MCP server"
	@echo "  clean  - Remove build artifacts"
	@echo "  format - Format Go code"
	@echo "  lint   - Run code linter"
	@echo ""
	@echo "Prerequisites: GEMINI_API_KEY environment variable"
