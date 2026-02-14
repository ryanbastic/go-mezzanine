 High Priority (must-have before production traffic)

  1. Authentication & Authorization

  You have zero auth. At minimum, add API key middleware or a reverse proxy (e.g., Envoy, nginx) in front. If the service is
  internal-only behind a mesh, service-to-service mTLS may suffice.

  2. TLS

  Either terminate TLS at a load balancer/proxy, or add tls.Config to http.Server. Don't run plain HTTP in production.

  3. Rate Limiting

  A token-bucket or sliding-window middleware (e.g., golang.org/x/time/rate) prevents a single caller from saturating the
  service.

  4. Containerization

  You have docker-compose.yml for Postgres but no Dockerfile for the server itself. A multi-stage build is straightforward:
  FROM golang:1.23 AS build
  WORKDIR /src
  COPY . .
  RUN CGO_ENABLED=0 go build -o /mezzanine ./cmd/mezzanine

  FROM gcr.io/distroless/static
  COPY --from=build /mezzanine /mezzanine
  ENTRYPOINT ["/mezzanine"]

  5. ~~Persistent Plugin Registry~~ ✅ Done

  The plugin registry is now backed by a PostgreSQL `plugins` table. Registrations are persisted on write and loaded
  from the database on startup.

  ---
  Medium Priority (operational maturity)

  6. Distributed Tracing

  Add OpenTelemetry spans. Propagate trace context through the request ID you already generate. This is critical for
  debugging cross-service issues.

  7. Circuit Breakers for Plugins

  Failed plugin calls retry with exponential backoff but have no circuit breaker. A plugin that's down will generate
  unbounded goroutines. Use something like sony/gobreaker.

  8. Query Timeouts

  Individual SQL queries don't have context deadlines. Pass ctx with a timeout from the request context so a hung query
  doesn't block forever.

  9. pprof Debug Endpoints

  Expose /debug/pprof behind auth for CPU/memory profiling in production. Cheap to add, invaluable for diagnosing issues.

  10. Alerting Rules & Dashboards

  You collect Prometheus metrics but have no example alerting rules or Grafana dashboards. Define SLOs (e.g., p99 < 50ms,
  error rate < 0.1%) and build alerts around them.

  ---
  Lower Priority (hardening & polish)

  - Pagination — partition/scan reads return unbounded results; add LIMIT/cursor
  - Index consistency — cell writes and index writes are separate operations with no transactional guarantee; consider
  wrapping in a Postgres transaction
  - Audit logging — record who wrote what and when for compliance
  - Load testing — use k6 or vegeta to find your breaking point before users do
  - Linting/CI — add .golangci.yml and a CI pipeline (GitHub Actions) for automated testing on PRs
  - Log correlation — include the request ID in all downstream log calls, not just the middleware

  ---
  Deployment Model

  The simplest production path:
  1. Dockerfile → push to a container registry
  2. Kubernetes Deployment with readiness/liveness probes pointing at your existing /v1/readyz and /v1/livez
  3. Postgres via a managed service (RDS, Cloud SQL, etc.) — not self-hosted
  4. Ingress/Load Balancer for TLS termination and routing
  5. Prometheus + Grafana scraping /metrics

  If Kubernetes is overkill for your scale, a systemd service behind nginx with Let's Encrypt works fine too.

  ---
  Want me to implement any of these? The quickest wins would be auth middleware, a Dockerfile, or persistent plugin storage.
