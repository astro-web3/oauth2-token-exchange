#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
SERVER_ADDR="${SERVER_ADDR:-http://localhost:8123}"
USER_ID="${USER_ID:-259242039378444290}"
EMAIL="${EMAIL:-test@example.com}"
PREFERRED_USERNAME="${PREFERRED_USERNAME:-testuser}"

echo -e "${GREEN}🧪 PAT Management E2E Test Suite${NC}"
echo "======================================"
echo ""
echo "Configuration:"
echo "  Server: $SERVER_ADDR"
echo "  User ID: $USER_ID"
echo "  Email: $EMAIL"
echo "  Preferred Username: $PREFERRED_USERNAME"
echo ""

# Check if server is running
echo -e "${YELLOW}Checking if server is running...${NC}"
if ! curl -s -f "$SERVER_ADDR/healthz" > /dev/null; then
    echo -e "${RED}❌ Server is not running at $SERVER_ADDR${NC}"
    echo "Please start the server first:"
    echo "  go run cmd/authz/main.go"
    exit 1
fi
echo -e "${GREEN}✅ Server is running${NC}"
echo ""

# Run the test
echo -e "${YELLOW}Running E2E tests...${NC}"
go run e2e/test-pat/main.go "$USER_ID" "$EMAIL" "$PREFERRED_USERNAME" "$SERVER_ADDR"

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}❌ Tests failed${NC}"
    exit 1
fi

