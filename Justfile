dev:
    @echo "Starting development server..."
    APP_ENV=local go run cmd/authz/main.go

gen:
    @echo "Generating code..."
    cd pb && buf generate

test:
    @echo "Running tests..."
    go test ./...