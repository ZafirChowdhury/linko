# Linko

A URL shortener built as a platform for learning production observability in Go. The core server was provided; the logging, metrics, tracing, and profiling layers were written as the learning exercise.

## Stack

| Concern | Tool |
|---|---|
| Structured logging | `log/slog` + `tint` (colored terminal), `lumberjack` (rotating files) |
| Metrics | Prometheus + Grafana |
| Distributed tracing | OpenTelemetry SDK → Jaeger (OTLP/gRPC) |
| Profiling | `net/http/pprof` (auth-gated) |
| Auth | HTTP Basic Auth + bcrypt |

## Running

**Prerequisites:** Go 1.26+, Docker

Start the observability stack:

```bash
docker compose up -d
```

Run the server:

```bash
go run . -port 8899 -data ./data
```

Optional — log to a rotating file in addition to stderr:

```bash
LINKO_LOG_FILE=./logs/linko.log go run . 
```

| Service | URL |
|---|---|
| Linko | http://localhost:8899 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 (admin/admin) |
| Jaeger UI | http://localhost:16686 |

## Endpoints

```
GET  /                    homepage
GET  /{shortCode}         redirect
POST /api/login           auth check        [basic auth]
POST /api/shorten         create short URL  [basic auth]
GET  /api/urls            list all URLs     [basic auth]
GET  /api/stats           redirect stats    [basic auth]
GET  /metrics             Prometheus scrape
GET  /debug/pprof/        pprof index       [basic auth]
```

Test users: `frodo` / `ofTheNineFingers`, `samwise` / `theStrong`

Load test scripts:

```bash
bash spamhomepage.sh          # 3500 GET / requests
bash spamredirect.sh 500      # 500 redirect requests
```

## What I Built

The server skeleton (routing, store, auth middleware structure) was provided. Everything related to observability was written from scratch:

- **Structured logging** — `slog.Logger` with dependency injection into the server; multi-handler setup writing colored output to stderr and JSON to a rotating file simultaneously.
- **Request logging middleware** — spy wrappers on `http.ResponseWriter` and `http.Request.Body` to capture status codes, bytes read/written, latency, request ID, and authenticated username per request.
- **Log context propagation** — `LogContext` stored on the request context so handlers can attach username and errors to the log entry written after the response is sent.
- **Security filtering** — `replaceAttr` hook redacts sensitive keys (`password`, `secret`, etc.) and strips passwords embedded in URL strings before they reach any log sink.
- **Error logging** — stack traces via `pkg/errors`, grouped attributes for multi-errors (`errors.Join`), and structured error groups so each field is individually queryable.
- **Log rotation** — `lumberjack` with 500 MB max size, 28-day retention, 10 backups, gzip compression.
- **Prometheus metrics** — `http_requests_total` counter vec labelled by method, path, and status; wired into a middleware before the route mux.
- **OpenTelemetry tracing** — tracer provider with OTLP/gRPC export to Jaeger; spans on every handler and the bcrypt/destination-check hot paths.
- **pprof** — profiling endpoints protected behind the same `authMiddleware` used for the API.

## What I Learned

From the [boot.dev](https://www.boot.dev/courses/learn-logging-observability-golang) observability course:

1. **Logs, metrics, and traces are complementary** — logs explain *what* happened, metrics show *how often*, traces show *where time went*.
2. **Structured logging at the boundary** — attaching context (request ID, user, build SHA) once in middleware means every downstream log line is automatically correlated without the handler knowing about it.
3. **Sensitive data leaks through logs** — URLs, error messages, and reflected form values are all vectors; a `ReplaceAttr` hook is the right single choke-point.
4. **bcrypt is intentionally slow** — the pprof CPU profile makes this obvious immediately; it dominated flamegraphs during load tests, which is exactly the point of profiling.
5. **Prometheus scrape interval matters** — 1 s scrape in dev gives responsive dashboards; in production, 15–30 s is the right trade-off between resolution and cardinality cost.
6. **Trace spans reveal hidden work** — the `checkDestination` HTTP call inside the redirect handler only became visible as a latency contributor once it had its own span.
