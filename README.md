# OAuth2 Token Exchange Service

Custom Authorization Service for Istio/Envoy ext_authz that exchanges Personal Access Tokens (PAT) with ZITADEL, caches the results, and provides PAT management APIs.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
  - [Authorization Flow](#authorization-flow)
  - [PAT Management APIs](#pat-management-apis)
- [Configuration](#configuration)
- [Quick Start](#quick-start)
- [API Endpoints](#api-endpoints)
- [Integration with Istio](#integration-with-istio)
- [Project Structure](#project-structure)
- [Technical Details](#technical-details)
  - [ZITADEL Token Exchange Flow](#zitadel-token-exchange-flow)
  - [Cache Strategy](#cache-strategy)
  - [Error Handling](#error-handling)
  - [Observability](#observability)
- [Deployment](#deployment)
- [Security Considerations](#security-considerations)

## Features

- **Authorization Service**: HTTP-based ext_authz for Istio/Envoy with PAT to JWT exchange
- **PAT Management**: gRPC/Connect-RPC APIs for creating, listing, and deleting Personal Access Tokens
- **Machine User Support**: Automatic machine user creation and token exchange with actor delegation
- **Redis Caching**: Token caching with configurable TTL to reduce ZITADEL API calls
  - Cache key: `authz:pat:<sha256(PAT)>`
  - Invalid tokens also cached to prevent cache penetration
- **Observability**: OpenTelemetry tracing and structured logging support
- **Graceful Shutdown**: Handles SIGINT/SIGTERM with timeout (10s)

## Architecture

### Authorization Flow

1. Receive HTTP requests from Istio/Envoy at `/oauth2/token-exchange/*` path
2. Extract PAT from `Authorization: Bearer <PAT>` header
3. Check Redis cache for cached JWT and user claims (key: `authz:pat:<sha256(PAT)>`)
4. If cache miss:
   - Call ZITADEL `/oidc/v1/userinfo` to get user info (username)
   - Use admin machine user PAT as `actor_token` to exchange user's username to JWT
   - Exchange via ZITADEL `/oauth/v2/token` with `grant_type=token-exchange`
   - Parse JWT claims (sub, email, groups, preferred_username)
   - Cache result in Redis with TTL
5. Return HTTP 200 OK with user headers injected (or 401/500 on error)

### PAT Management APIs

Provides Connect-RPC service (`pat.v1.PATService`) for:

- **CreatePAT**: Create machine users and generate PATs for them
- **ListPATs**: List all PATs for a machine user
- **DeletePAT**: Revoke a PAT by ID

## Configuration

Edit `config/config.yaml` or `config/config.local.yaml`:

```yaml
server:
  addr: ":8123"
  mode: "release"            # "release" or "debug"
  read_timeout: 30s
  write_timeout: 30s

redis:
  url: "redis://localhost:6379/0"
  pool_size: 50

auth:
  admin_machine_user:
    pat: ""                  # Admin PAT for token exchange with actor delegation
  zitadel:
    issuer: "https://auth.modellink.ai"
    client_id: "your-client-id"
    client_secret: "your-client-secret"
    organization_id: ""      # For creating machine users
  cache_ttl: 5m
  header_keys:
    user_id: "X-Auth-Request-User"
    user_email: "X-Auth-Request-Email"
    user_groups: "X-Auth-Request-Groups"
    user_preferred_username: "X-Auth-Request-Preferred-Username"
    user_jwt: "X-Auth-Request-Access-Token"

observability:
  metrics_enabled: false
  trace_enabled: false
  tracing_endpoint_url: ""   # e.g., "http://localhost:4318"
  log_level: "info"          # "debug", "info", "warn", "error"
  log_format: "json"         # "json" or "text"
  log_source: false
```

### Environment Variables

All config values can be overridden via environment variables with prefix `OAUTH2_TOKEN_EXCHANGE_`:

```bash
OAUTH2_TOKEN_EXCHANGE_SERVER_ADDR=:8123
OAUTH2_TOKEN_EXCHANGE_REDIS_URL=redis://localhost:6379/0
OAUTH2_TOKEN_EXCHANGE_AUTH_ADMIN_MACHINE_USER_PAT=<admin-pat>
OAUTH2_TOKEN_EXCHANGE_AUTH_ZITADEL_ISSUER=https://auth.modellink.ai
```

## Quick Start

### Development

Use [Just](https://github.com/casey/just) commands:

```bash
# Run in development mode (loads config.local.yaml)
just dev

# Generate Protobuf code
just gen

# Run tests
just test

# Run linter
just lint

# Auto-fix linting issues
just fix
```

### Building

```bash
go build -o authz ./cmd/authz
```

Or build Docker image:

```bash
just docker-build
```

### Running

```bash
# Run binary
./authz

# Run with custom config
APP_ENV=local ./authz

# Run with Docker
just docker-run
```

## API Endpoints

### Health Check

```bash
GET /healthz
# Returns: "ok" (200)
```

### Authorization Endpoint (for Istio ext_authz)

```bash
ANY /oauth2/token-exchange/*
Authorization: Bearer <PAT>

# Response Headers (on 200 OK):
X-Auth-Request-User: <user_id>
X-Auth-Request-Email: <email>
X-Auth-Request-Groups: <group1,group2>
X-Auth-Request-Preferred-Username: <username>
X-Auth-Request-Access-Token: <jwt>
```

### PAT Management APIs (Connect-RPC)

Base path: `/pat.v1.PATService/*`

```protobuf
service PATService {
  // Create machine user and generate PAT
  rpc CreatePAT(CreatePATRequest) returns (CreatePATResponse);
  
  // List all PATs for a machine user
  rpc ListPATs(ListPATsRequest) returns (ListPATsResponse);
  
  // Revoke a PAT
  rpc DeletePAT(DeletePATRequest) returns (DeletePATResponse);
}
```

#### Example: Create PAT

```bash
# gRPC/Connect-RPC call
curl -X POST https://your-domain/pat.v1.PATService/CreatePAT \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-pat>" \
  -d '{
    "user_id": "user123",
    "email": "user@example.com",
    "preferred_username": "machine_user_123",
    "expiration_date": "2025-12-31T23:59:59Z"
  }'
```

## Integration with Istio

### Configure Extension Provider

Add to your `IstioOperator` or Helm values:

```yaml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  meshConfig:
    extensionProviders:
    - name: "oauth2-token-exchange"
      envoyExtAuthzHttp:
        service: "oauth2-token-exchange.authz.svc.cluster.local"
        port: "8123"
        pathPrefix: "/oauth2/token-exchange"
        headersToDownstreamOnAllow:
          - X-Auth-Request-User
          - X-Auth-Request-Email
          - X-Auth-Request-Groups
          - X-Auth-Request-Preferred-Username
          - X-Auth-Request-Access-Token
```

### Apply Authorization Policy

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: ext-authz-pat
  namespace: your-namespace
spec:
  action: CUSTOM
  provider:
    name: "oauth2-token-exchange"
  rules:
    - to:
      - operation:
          paths: ["/api/*"]  # Apply to specific paths
```

### How It Works

1. **Istio forwards request**: Envoy sends the original request (with Authorization header) to this service
2. **Service validates PAT**: 
   - Extracts PAT from `Authorization: Bearer <PAT>`
   - Checks Redis cache or exchanges with ZITADEL
   - Invalid/expired PATs are cached to prevent cache penetration
3. **Return decision**:
   - **200 OK** + User headers → Istio allows request and injects headers
   - **401 Unauthorized** → Istio denies request
   - **500 Internal Server Error** → Istio denies request
4. **Downstream service receives**:
   - Original request headers
   - Injected user headers (`X-Auth-Request-*`)
   - Can use these headers for user identification

## Project Structure

Follows **Clean Architecture** with dependency inversion:

```
oauth2-token-exchange/
├── cmd/authz/              # Entry point (main.go)
├── config/                 # YAML configuration files
├── internal/
│   ├── app/                # Application layer (orchestration)
│   │   ├── authz/          # Authorization service
│   │   └── pat/            # PAT management (CQRS: command + query)
│   ├── config/             # Config loading (viper)
│   ├── domain/             # Domain layer (business logic)
│   │   ├── authz/          # Authorization domain
│   │   │   ├── service.go  # PAT exchange logic
│   │   │   └── types.go    # AuthzDecision, TokenClaims
│   │   └── pat/            # PAT management domain
│   │       ├── service.go  # PAT CRUD logic
│   │       ├── repo.go     # Repository interface
│   │       └── entity.go   # PAT entity
│   ├── infra/              # Infrastructure layer (implementations)
│   │   ├── cache/          # Redis client + TokenCache interface
│   │   └── zitadel/        # ZITADEL API client (token exchange, userinfo, PAT CRUD)
│   └── transport/          # Transport layer (HTTP/gRPC handlers)
│       └── http/
│           ├── router.go   # Gin router setup
│           ├── handler.go  # Authorization check handler
│           └── middleware.go # Logging middleware
├── pb/                     # Protobuf definitions
│   ├── pat/v1/             # PAT service proto
│   ├── envoy/              # Envoy ext_authz (not used, HTTP mode)
│   └── gen/                # Generated code (Go + OpenAPI)
└── pkg/                    # Reusable utilities
    ├── http/               # HTTP client wrapper
    ├── logger/             # Structured logging (slog)
    ├── otel/               # OpenTelemetry setup
    └── tracer/             # Tracing helpers
```

### Architecture Principles

- **Dependency Inversion**: Domain defines interfaces, Infra implements them
- **CQRS**: PAT management split into Command (write) and Query (read) services
- **Layered**:
  - `Transport` → `App` → `Domain` ← `Infra`
  - `Domain` has no dependencies on outer layers

## Technical Details

### ZITADEL Token Exchange Flow

This service uses ZITADEL's OAuth2 Token Exchange (RFC 8693) with actor delegation:

1. **Get User Info**: Call `/oidc/v1/userinfo` with user's PAT to get `preferred_username`
2. **Exchange with Actor**: Call `/oauth/v2/token` with:
   - `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`
   - `subject_token=<username>` (from step 1)
   - `subject_token_type=urn:zitadel:params:oauth:token-type:user_id`
   - `actor_token=<admin_machine_user_pat>` (from config)
   - `actor_token_type=urn:ietf:params:oauth:token-type:access_token`
3. **Parse JWT**: Extract claims from returned `id_token`

**Why Actor Delegation?**
- User's PAT cannot be directly exchanged to JWT (ZITADEL limitation)
- Admin machine user acts as "actor" to impersonate the user
- Returns a valid JWT with user's claims

### Cache Strategy

- **Cache Key**: SHA-256 hash of PAT (`authz:pat:<hex(sha256(PAT))>`)
- **Cache Value**: JSON containing:
  ```json
  {
    "access_token": "<jwt>",
    "user_id": "<sub>",
    "email": "<email>",
    "groups": ["group1", "group2"],
    "preferred_username": "<username>",
    "is_invalid": false
  }
  ```
- **Invalid Token Caching**: If exchange fails, cache `{"is_invalid": true}` to prevent repeated invalid requests
- **TTL**: Configurable (default 5 minutes)

### Error Handling

| Scenario | Response | Cached |
|----------|----------|--------|
| Empty Authorization header | 200 OK (no headers) | No |
| Invalid PAT | 401 Unauthorized | Yes (`is_invalid=true`) |
| ZITADEL API error | 401 Unauthorized | Yes (`is_invalid=true`) |
| Redis error | 500 Internal Server Error | No |
| Token exchange success | 200 OK + headers | Yes |

### Observability

**Tracing Spans** (OpenTelemetry):
- `transport.http.Check`: HTTP handler
- `app.authz.Check`: Application layer
- `domain.authz.AuthorizePAT`: Domain layer

**Logging** (structured with slog):
- Request ID (from OpenTelemetry trace)
- PAT prefix (first 8 chars for debugging)
- Authorization decision (allow/deny + reason)
- Errors with full context

## Deployment

### Kubernetes Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oauth2-token-exchange
  namespace: authz
spec:
  replicas: 2
  selector:
    matchLabels:
      app: oauth2-token-exchange
  template:
    metadata:
      labels:
        app: oauth2-token-exchange
    spec:
      containers:
      - name: authz
        image: oauth2-token-exchange:latest
        ports:
        - containerPort: 8123
        env:
        - name: APP_ENV
          value: "prod"
        - name: OAUTH2_TOKEN_EXCHANGE_REDIS_URL
          value: "redis://redis.authz.svc.cluster.local:6379/0"
        - name: OAUTH2_TOKEN_EXCHANGE_AUTH_ADMIN_MACHINE_USER_PAT
          valueFrom:
            secretKeyRef:
              name: zitadel-admin
              key: pat
        - name: OAUTH2_TOKEN_EXCHANGE_AUTH_ZITADEL_ISSUER
          value: "https://auth.modellink.ai"
        - name: OAUTH2_TOKEN_EXCHANGE_AUTH_ZITADEL_CLIENT_ID
          valueFrom:
            secretKeyRef:
              name: zitadel-client
              key: client_id
        - name: OAUTH2_TOKEN_EXCHANGE_AUTH_ZITADEL_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: zitadel-client
              key: client_secret
        volumeMounts:
        - name: config
          mountPath: /app/config
      volumes:
      - name: config
        configMap:
          name: oauth2-token-exchange-config
---
apiVersion: v1
kind: Service
metadata:
  name: oauth2-token-exchange
  namespace: authz
spec:
  ports:
  - port: 8123
    targetPort: 8123
  selector:
    app: oauth2-token-exchange
```

## Security Considerations

- **PAT Hashing**: PATs are hashed with SHA-256 before using as Redis keys (never store plaintext)
- **Admin PAT**: Store admin machine user PAT in Kubernetes Secret, not in config files
- **TLS**: Use Istio mTLS for service-to-service communication
- **Rate Limiting**: Consider adding rate limiting on `/oauth2/token-exchange/*` endpoint
- **Cache TTL**: Balance between performance and security (shorter TTL = more secure but more API calls)

## License

MIT

