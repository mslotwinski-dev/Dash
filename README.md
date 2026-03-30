# Dash

**Dash** is a custom reverse proxy project written in Go, inspired by tools like Caddy and NGINX.  
Its goal is to provide a production-ready HTTP/HTTPS proxy with modern traffic management, security, observability, and cloud-friendly deployment workflows.

> Project status: **early stage / documentation-first roadmap**.  
> This repository currently defines the vision and implementation path for a full-featured reverse proxy platform.

---

## Table of Contents

- [Why Dash](#why-dash)
- [Core Objectives](#core-objectives)
- [Feature Scope](#feature-scope)
- [Architecture (Target State)](#architecture-target-state)
- [Development Roadmap](#development-roadmap)
- [Cloud Engineering Extension](#cloud-engineering-extension)
- [Getting Started](#getting-started)
- [Configuration Direction](#configuration-direction)
- [Observability Direction](#observability-direction)
- [Security Direction](#security-direction)
- [Testing and Quality Strategy](#testing-and-quality-strategy)
- [Deployment Direction](#deployment-direction)
- [Contributing](#contributing)
- [License](#license)

---

## Why Dash

Dash is designed as a practical, end-to-end engineering project that combines:

- reverse proxy fundamentals,
- transport security (TLS/SSL),
- load balancing algorithms,
- performance optimizations (compression and caching),
- operations tooling (logs and metrics),
- and optional real-time administration capabilities.

It is intended both as:

1. a useful infrastructure component, and  
2. a strong portfolio project for backend and cloud engineering growth.

---

## Core Objectives

Dash aims to deliver:

1. **Reverse proxying for HTTP/HTTPS traffic**
2. **Automatic SSL certificate management** (e.g., Let's Encrypt)
3. **Load balancing** with multiple strategies
4. **Response optimization** (gzip/brotli + static cache)
5. **Operational visibility** (structured logs + Prometheus metrics)
6. **Advanced traffic control** (path/header routing, rate limiting, IP filtering)
7. **Extensibility** for WebSockets, sticky sessions, and dynamic backend management

---

## Feature Scope

### 1) Reverse Proxy and Routing

- Forward incoming client requests to configured backend services
- Support routing by:
  - URL path (e.g., `/api`, `/static`)
  - HTTP headers (e.g., `User-Agent`, `X-Tenant`) *(bonus / advanced)*

### 2) Load Balancing

- **Round-robin**: cycle through backend pool
- **Least connections**: choose backend with smallest active connection count
- **Weighted balancing**: distribute requests proportionally to backend weights

### 3) TLS / HTTPS

- HTTPS listener with TLS termination
- Automatic certificate issuance and renewal (Let's Encrypt via `autocert`)
- Optional HTTP → HTTPS redirect

### 4) Compression and Caching

- Compression based on `Accept-Encoding`:
  - `gzip`
  - `br` (brotli)
- Static response caching:
  - in-memory cache with TTL
  - cache cleanup cycle
  - optional `ETag` and `Cache-Control` behavior

### 5) Logging and Monitoring

- HTTP access logs:
  - method, URL, status, latency
- Error and panic logging (`defer` + `recover`)
- Metrics endpoint (`/metrics`) for Prometheus:
  - request count
  - response duration histogram
  - status code distribution

### 6) Security Controls

- Rate limiting (token bucket / leaky bucket style)
- IP allowlist / blocklist
- Basic anti-DDoS behavior through traffic shaping and request controls

### 7) Advanced/Optional Features

- WebSocket proxy support (Upgrade/Connection flow)
- Sticky sessions (session affinity)
- Dynamic backend updates without process restart
- Admin panel (real-time status with WebSockets)

---

## Architecture (Target State)

```text
Client
  |
  | HTTP/HTTPS
  v
Dash Reverse Proxy
  ├─ TLS Termination (autocert / cert store)
  ├─ Request Logging + Metrics Middleware
  ├─ Security Layer (Rate Limit, IP Filters)
  ├─ Router (Path/Header Rules)
  ├─ Load Balancer (RR / LeastConn / Weighted)
  ├─ Compression + Cache
  └─ Upstream Transport
       ├─ Backend A
       ├─ Backend B
       └─ Backend C
```

Optional control plane components:

- Config watcher/API for dynamic backend changes
- Admin UI + WebSocket stream for runtime state
- External observability stack (Prometheus/Grafana/CloudWatch)

---

## Development Roadmap

The project can be implemented incrementally:

### Stage 0 — Environment Setup

- Go, Docker, Git setup
- Repository bootstrap (`go.mod`, `.gitignore`, optional `Makefile`)

### Stage 1 — MVP Proxy

- Basic HTTP reverse proxy
- Backend list and round-robin
- Request logging (method, URL, status, latency)

### Stage 2 — HTTPS and Certificates

- TLS listener
- Let's Encrypt automation
- Optional HTTP → HTTPS redirect

### Stage 3 — Config from File

- YAML/JSON config for backend list and weights
- Hot-reload on config changes

### Stage 4 — Advanced Load Balancing

- Least-connections
- Weighted round-robin

### Stage 5 — Compression + Cache

- gzip/brotli support
- TTL cache for static or cacheable responses
- Optional ETag and Cache-Control integration

### Stage 6 — Monitoring and Logging

- Log files + robust error logging
- Prometheus metrics endpoint

### Stage 7 — Smart Routing

- Route by path and selected headers

### Stage 8 — Traffic Protection

- Per-IP rate limits
- IP allow/block rules

### Stage 9 — WebSockets + Sticky Sessions

- WebSocket proxying
- Session affinity

### Stage 10 — Admin Panel (Bonus)

- Real-time backend health view
- Add/remove backend operations via API/WebSocket

### Stage 11 — Deployment and Documentation

- Docker + Compose
- Cloud deployment
- Complete usage and operations docs

---

## Cloud Engineering Extension

To turn Dash into a realistic cloud engineering project, pair each local feature with managed cloud services:

| Area | Local Implementation | Cloud-Grade Extension |
|---|---|---|
| Proxy Runtime | Go process | EC2, ECS, or Kubernetes workload |
| TLS | Let's Encrypt autocert | ACM + DNS validation (Route53) |
| Load Balancing | In-app LB logic | ALB/NLB in front of Dash |
| Static Delivery | In-proxy cache | CloudFront + S3 origin |
| Config | Local YAML | S3/Parameter Store/Secrets Manager |
| Metrics & Logs | Prometheus endpoint + files | Prometheus + CloudWatch/Grafana |
| CI/CD | Local build | GitHub Actions + container registry + deploy pipeline |
| Infrastructure | Manual setup | Terraform for VPC, SG, compute, DNS, certs |

Recommended minimum cloud path:

1. Containerize Dash
2. Deploy to AWS (EC2 or ECS/Fargate)
3. Configure domain + TLS
4. Provision infrastructure with Terraform
5. Expose metrics and centralize logs

---

## Getting Started

### Prerequisites

- Go 1.22+
- Git
- Docker (recommended for containerized workflows)

### Clone

```bash
git clone https://github.com/mslotwinski-dev/Dash.git
cd Dash
```

### Current Repository State

This repository currently contains the module skeleton and project documentation.  
Implementation code can be developed incrementally according to the roadmap above.

---

## Configuration Direction

Planned configuration model includes:

- listener settings (HTTP/HTTPS ports)
- certificate settings (email, domain policy, cert storage)
- backend definitions (address, weight, health metadata)
- load balancing mode
- routing rules (path and header matches)
- cache and compression toggles
- security limits (rate limit, IP lists)
- observability settings (log level/format, metrics endpoint)

---

## Observability Direction

Dash should expose:

- **Access logs** with request metadata and latency
- **Error logs** including panic recovery context
- **Prometheus metrics** for traffic, latency, and failures
- Optional backend health data for admin UI/automation

This enables:

- incident analysis,
- performance tuning,
- and capacity planning.

---

## Security Direction

Minimum security baseline:

- TLS by default for public traffic
- Rate limiting per source IP
- IP allow/deny controls
- Safe handling of forwarded headers
- Request timeout strategy to reduce resource exhaustion risk

As the project matures, security can be expanded with:

- WAF integration,
- mTLS upstream,
- signed admin actions,
- and secret management through cloud KMS/secret stores.

---

## Testing and Quality Strategy

Recommended quality gates as implementation progresses:

- Unit tests for balancing/routing/cache logic
- Integration tests with multiple mock backends
- TLS and cert-flow validation in staging
- Load tests for throughput and latency under pressure
- Failure tests (backend down, timeout spikes, invalid certs)

---

## Deployment Direction

Suggested production packaging:

- `Dockerfile` for Dash runtime
- `docker-compose.yml` for local multi-service testing
- IaC (Terraform) for cloud environments
- CI/CD pipeline for automated build, test, and rollout

---

## Contributing

Contributions are welcome.  
When contributing:

1. Keep changes scoped and focused
2. Add tests for behavior changes
3. Update documentation for new features
4. Follow roadmap stage boundaries when possible

---

## License

This project is licensed under the terms of the [MIT License](LICENSE).
