# azugo

Opinionated GoLang web framework for microservices based on FastHTTP.

### Features

* HTTP web server [valyala/fasthttp](https://github.com/valyala/fasthttp)
* HTTP/2 support [forked dgrr/http2](https://github.com/lafriks/http2)
* Web socket support [dgrr/websocket](https://github.com/dgrr/websocket)
* Structured logger [go.uber.org/zap](https://github.com/uber-go/zap)
* JSON serialization [goccy/go-json](https://github.com/goccy/go-json)
* Data structure validation using [go-playground/validator](https://github.com/go-playground/validator)
* Built-in web app testing framework

### Special Environment variables used by the Azugo framework

#### Core

* `ENVIRONMENT` - App environment setting (allowed values are `development`, `test`, `staging` and `production`).
* `LOG_TYPE` - Log type (defaults to `console`, allowed values are `console`, `file`, `otel` (provided by [azugo.io/opentelemetry](https://pkg.go.dev/azugo.io/opentelemetry)) or other registered log drivers).
* `LOG_LEVEL` - Minimal log level (defaults to `info`, allowed values are `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, `fatal`).
* `LOG_FORMAT` - Log output format (defaults to `console` in development environment and `ecsjson` in staging and production).
* `LOG_OUTPUT` - Log output location (defaults to `stderr`, allowed values are `stderr`, `stdout`, file path or `file://` URL and other values supported by registered log drivers).
* `LOG_STACKTRACE` - Enable stack traces for error level and above regardless of environment (defaults to `false`).
* `LOG_TYPE_SECONDARY` - Secondary log type (see `LOG_TYPE`).
* `LOG_LEVEL_SECONDARY` - Secondary log level (defaults to `info`, see `LOG_LEVEL`).
* `LOG_FORMAT_SECONDARY` - Secondary log format (see `LOG_FORMAT`).
* `LOG_OUTPUT_SECONDARY` - Secondary log output location (see `LOG_OUTPUT`).

#### Server

* `SERVER_URLS` - Server URL or multiple URLs separated by semicolons to listen on.
* `SERVER_HTTPS_CERTIFICATE_PEM_FILE` - Path to PEM file for HTTPS certificate.
* `SERVER_READ_TIMEOUT` - Maximum duration for reading the full request including body (defaults to `30s`).
* `SERVER_WRITE_TIMEOUT` - Maximum duration for writing the response (defaults to `10s`).
* `SERVER_IDLE_TIMEOUT` - Maximum duration to wait for the next request on a keep-alive connection (defaults to `75s`).
* `SERVER_MAX_REQUEST_BODY_SIZE` - Maximum request body size in bytes (defaults to `4194304` (4MB)).
* `SERVER_SHUTDOWN_TIMEOUT` - Maximum duration to wait for active connections to finish on graceful shutdown (defaults to `30s`).
* `BASE_PATH` - Base path for the app if deployed in a subdirectory.
* `ACCESS_LOG_ENABLED` - Enable access logs (defaults to `true`).

#### Reverse Proxy

* `REVERSE_PROXY_TRUSTED_IPS` - Semicolon-separated list of trusted proxy IP addresses or CIDR ranges (defaults to `127.0.0.1`).
* `REVERSE_PROXY_TRUSTED_HEADERS` - Semicolon-separated list of trusted proxy headers (defaults to `X-Real-IP;X-Forwarded-For`).
* `REVERSE_PROXY_LIMIT` - Maximum number of trusted proxies in chain (defaults to `1`).

#### CORS

* `CORS_ORIGINS` - Semicolon-separated list of allowed CORS origins.

#### Metrics

* `METRICS_ENABLED` - Enable Prometheus metrics endpoint (defaults to `true`).
* `METRICS_PATH` - Metrics endpoint path (defaults to `/metrics`).
* `METRICS_TRUSTED_IPS` - Semicolon-separated list of trusted IP addresses or CIDR ranges allowed to access the metrics endpoint (defaults to `127.0.0.1`).

#### Health Check

* `HEALTHZ_ENABLED` - Enable health check endpoint (defaults to `true`).
* `HEALTHZ_TRUSTED_IPS` - Semicolon-separated list of trusted IP addresses or CIDR ranges allowed to access the health check endpoint (defaults to all loopback and private network ranges: `127.0.0.0/8`, `::1/128`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `fc00::/7`).

#### Pagination

* `PAGING_DEFAULT_PAGE_SIZE` - Default page size for paginated responses (defaults to `20`).
* `PAGING_MAX_PAGE_SIZE` - Maximum allowed page size for paginated responses (defaults to `100`).

#### Cache

* `CACHE_TYPE` - Cache type to use (defaults to `memory`, allowed values are `memory`, `redis`, `redis-cluster`, `redis-sentinel`).
* `CACHE_TTL` - Duration to keep items in cache (defaults to `0` meaning never expire).
* `CACHE_KEY_PREFIX` - Prefix all cache keys with specified value.
* `CACHE_CONNECTION` - Connection string for non-memory cache backends.
* `CACHE_PASSWORD` - Password for cache connection.
* `CACHE_PASSWORD_FILE` - File to read value for `CACHE_PASSWORD` from.

### Special thanks to

* Router largely based on [fasthttp/router](https://github.com/fasthttp/router)
