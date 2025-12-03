# OAuth2 Token Exchange Authz Service

Custom Authorization Service for Istio/Envoy ext_authz that exchanges Personal Access Tokens (PAT) with ZITADEL and caches the results.

## Architecture

This service implements the Envoy ext_authz protocol to:

1. Receive `CheckRequest` from Istio/Envoy
2. Extract PAT from `Authorization: Bearer <PAT>` header
3. Check Redis cache for cached JWT and user claims
4. If cache miss, exchange PAT with ZITADEL `/oauth/v2/token` endpoint
5. Get userinfo from ZITADEL `/oidc/v1/userinfo` endpoint
6. Cache the result in Redis with TTL (default 5 minutes)
7. Return `CheckResponse` with user headers injected

## Configuration

Edit `config/config.yaml` or `config/config.local.yaml`:

```yaml
server:
  addr: ":8080"
  mode: "release"

redis:
  url: "redis://localhost:6379/0"
  pool_size: 10

auth:
  zitadel:
    issuer: "https://auth.modellink.ai"
    client_id: "your-client-id"
    client_secret: "your-client-secret"
  cache_ttl: 5m
  header_keys:
    user_id: "x-user-id"
    user_email: "x-user-email"
    user_groups: "x-user-groups"
    user_jwt: "x-user-jwt"

observability:
  metrics_enabled: true
  trace_enabled: true
  tracing_endpoint_url: "http://localhost:4318"
  log_level: "info"
```

## Building

```bash
go build -o bin/authz ./cmd/authz
```

## Running

```bash
./bin/authz
```

Or with environment variables:

```bash
OAUTH2_AUTHZ_SERVER_ADDR=:8080 \
OAUTH2_AUTHZ_REDIS_URL=redis://localhost:6379/0 \
./bin/authz
```

## Testing

```bash
go test ./...
```

## Integration with Istio

Configure Istio AuthorizationPolicy to use this service:

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: ext-authz
spec:
  action: CUSTOM
  provider:
    name: "oauth2-authz"
  rules:
    - {}
```

And configure the extension provider:

```yaml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  meshConfig:
    extensionProviders:
    - name: "oauth2-authz"
      envoyExtAuthzGrpc:
        service: "oauth2-token-exchange.authz.svc.cluster.local"
        port: "8080"
```

## Project Structure

```
oauth2-token-exchange/
├── cmd/authz/          # Main entry point
├── config/             # Configuration files
├── internal/
│   ├── app/authz/      # Application layer
│   ├── config/         # Config loading
│   ├── domain/authz/    # Domain layer
│   ├── infra/
│   │   ├── cache/      # Redis cache implementation
│   │   └── zitadel/    # ZITADEL client
│   └── transport/grpc/ # gRPC/Connect handlers
├── pb/                 # Protobuf definitions
│   ├── envoy/          # Envoy ext_authz proto
│   └── common/         # Common types
└── pkg/                # Shared packages
    ├── http/           # HTTP client
    ├── logger/         # Logging
    └── tracer/         # Tracing
```

