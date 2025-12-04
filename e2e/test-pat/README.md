# PAT Management E2E Tests

End-to-end tests for Personal Access Token (PAT) management functionality.

## Prerequisites

1. Start the oauth2-token-exchange server:
   ```bash
   go run cmd/authz/main.go
   ```

2. Ensure Redis is running (for token caching)

3. Configure `config/config.local.yaml` with:
   - Admin machine user PAT
   - Zitadel issuer, client ID, and client secret

## Running the Tests

### Option 1: Using the test script (Recommended)

```bash
cd oauth2-token-exchange
./e2e/test-pat/test.sh
```

Or with custom configuration:

```bash
USER_ID="259242039378444290" \
EMAIL="user@example.com" \
PREFERRED_USERNAME="testuser" \
SERVER_ADDR="http://localhost:8123" \
./e2e/test-pat/test.sh
```

### Option 2: Direct execution

```bash
go run e2e/test-pat/main.go <user-id> <email> <preferred-username> [server-addr]
```

### Example

```bash
go run e2e/test-pat/main.go \
  "259242039378444290" \
  "user@example.com" \
  "testuser" \
  "http://localhost:8123"
```

## Test Coverage

The E2E test suite covers:

1. **CreatePAT** - Creates a new PAT with custom expiration time
2. **ListPATs** - Lists all PATs for a user
3. **AuthorizePAT** - Tests authorization using the created PAT
4. **DeletePAT** - Deletes the created PAT

## Expected Output

```
🧪 Starting PAT Management E2E Tests
=====================================

📝 Test: CreatePAT
   Created PAT ID: <pat-id>
   Machine User ID: <machine-user-id>
   Human User ID: <human-user-id>
   Expiration Date: <expiration>
✅ CreatePAT test passed (PAT ID: <pat-id>)

📋 Test: ListPATs
   Found 1 PAT(s)
   PAT 1: ID=<pat-id>, MachineUserID=<machine-user-id>, HumanUserID=<human-user-id>, Expires=<expiration>
✅ ListPATs test passed

🔐 Test: AuthorizePAT
   Authorization: ALLOWED
   User ID: <user-id>
   Email: <email>
   Access Token: <token-preview>...
✅ AuthorizePAT test passed

🗑️  Test: DeletePAT
   Deleted PAT ID: <pat-id>
✅ DeletePAT test passed

🎉 All E2E tests passed!
```

