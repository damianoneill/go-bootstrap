# Server Customization Example

This example demonstrates how to set up a production-grade HTTPS server with Go Bootstrap, including:

- TLS/HTTPS configuration with modern security settings
- HTTP/2 support
- Custom security headers
- Server pre-start customization
- Rate limiting
- TLS debugging capabilities

## Middleware Configuration

The example demonstrates structured middleware ordering with categories:

1. Security Middleware (runs first)
   - Security headers (X-Content-Type-Options, X-Frame-Options, X-XSS-Protection)
   - Core security middlewares provided by the bootstrap library

2. Core Middleware
   - Request ID generation
   - Real IP detection
   - Panic recovery
   - Base timeouts

3. Application Middleware
   - Rate limiting (10 req/s per client)

4. Observability Middleware (runs last)
   - Request logging
   - Metrics collection
   - Distributed tracing

This ordering ensures security checks are performed first, followed by core HTTP handling, application-specific logic, and finally observability instrumentation.

## Getting Started

1. Generate self-signed certificates for development:

```bash
./gen-certs.sh
```

This script generates:

- `certs/ca.key` and `certs/ca.crt`: Development CA certificate
- `certs/server.key` and `certs/server.crt`: Server certificate signed by the CA

2. Run the server:

```bash
go run main.go
```

## Testing TLS Configuration

1. Basic HTTPS request:

```bash
curl -k https://localhost:9443/api/v1/secure
```

2. Detailed TLS debug information:

```bash
curl -k https://localhost:9443/api/v1/tls-debug
```

3. Test TLS handshake with OpenSSL:

```bash
openssl s_client -connect localhost:9443 -tls1_2
```

The `-tls1_2` flag forces TLS 1.2. The server also supports TLS 1.3 for improved security.

## Security Features

1. TLS Configuration:
   - Minimum TLS version: 1.2
   - Modern cipher suites prioritizing AES-GCM and ChaCha20
   - HTTP/2 support
   - Certificate verification

2. Security Headers:
   - X-Content-Type-Options: nosniff
   - X-Frame-Options: DENY
   - X-XSS-Protection: 1; mode=block

3. Rate Limiting:
   - 10 requests per second per client
   - Maximum backlog of 50 requests

## Available Endpoints

- `/api/v1/secure` - Basic secure endpoint
- `/api/v1/tls-debug` - Shows TLS connection details
- `/internal/health` - Health check endpoint
- `/internal/ready` - Readiness check endpoint
- `/internal/metrics` - Prometheus metrics

## Configuration

All server settings can be customized via the `config.yaml` file or environment variables:

```bash
SRV_EXAMPLE_SERVER_HTTP_PORT=9443 ./server-customization
```

## Production Considerations

For production use:

1. Replace self-signed certificates with proper CA-signed certificates
2. Consider enabling client certificate verification for mutual TLS
3. Review and adjust rate limiting settings based on your needs
4. Configure appropriate timeouts for your use case
5. Consider adding additional security headers based on your requirements

## Debugging

The `/api/v1/tls-debug` endpoint provides detailed information about the TLS connection:

- TLS version
- Negotiated cipher suite
- Protocol (HTTP/1.1 or HTTP/2)
- Certificate details
- Client connection information

For more detailed TLS debugging, use the openssl client:

```bash
openssl s_client -connect localhost:9443 -tls1_2 -msg -debug
```
