dev:
    @echo "Starting development server..."
    APP_ENV=local go run cmd/authz/main.go

gen:
    @echo "Generating code..."
    cd pb && buf generate

test:
    @echo "Running tests..."
    go test ./...

docker-build:
    @echo "Building Docker image..."
    docker build -t oauth2-token-exchange:latest .

docker-run:
    @echo "Running Docker image..."
    docker run --rm -p 8123:8123 -v $(pwd)/config:/app/config -e APP_ENV=local oauth2-token-exchange:latest

lint:
    @echo "Running golangci-lint..."
    golangci-lint run

fix:
    @echo "Auto-fixing linting issues..."
    golangci-lint run --fix