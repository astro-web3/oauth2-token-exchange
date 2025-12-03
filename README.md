# OAuth2 Token Exchange Authz Service

Custom Authorization Service for Istio/Envoy ext_authz that exchanges Personal Access Tokens (PAT) with ZITADEL and caches the results.

## Architecture

This service implements HTTP-based authorization for Istio/Envoy ext_authz:

1. Receive HTTP requests from Istio/Envoy at `/oauth2/token-exchange/*` path
2. Extract PAT from `Authorization: Bearer <PAT>` header
3. Check Redis cache for cached JWT and user claims
4. If cache miss, exchange PAT with ZITADEL `/oauth/v2/token` endpoint
5. Get userinfo from ZITADEL `/oidc/v1/userinfo` endpoint
6. Cache the result in Redis with TTL (default 5 minutes)
7. Return HTTP 200 OK with user headers injected (or 401/500 on error)

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

And configure the extension provider to use HTTP mode:

```yaml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  meshConfig:
    extensionProviders:
    - name: "oauth2-authz"
      envoyExtAuthzHttp:
        service: "oauth2-token-exchange.authz.svc.cluster.local"
        port: "8080"
        pathPrefix: "/oauth2/token-exchange"
```

### How It Works

Istio ext-authz HTTP mode forwards the original HTTP request (Method + Path) to the authorization service. This service:

- Accepts any HTTP method and path under `/oauth2/token-exchange/*`
- Extracts the `Authorization: Bearer <PAT>` header from the request
- Returns HTTP 200 OK with user headers if authorization succeeds
- Returns HTTP 401 Unauthorized if authorization fails
- Returns HTTP 500 Internal Server Error on service errors

Istio checks the HTTP status code:
- **200 OK**: Request is allowed, headers are injected into the original request
- **4xx/5xx**: Request is denied

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
│   └── transport/http/ # HTTP/Gin handlers
├── pb/                 # Protobuf definitions
│   ├── envoy/          # Envoy ext_authz proto
│   └── common/         # Common types
└── pkg/                # Shared packages
    ├── http/           # HTTP client
    ├── logger/         # Logging
    └── tracer/         # Tracing
```

