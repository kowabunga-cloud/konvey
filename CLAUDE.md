# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Konvey is the **Kowabunga Network Load-Balancer Agent** — a Linux daemon that dynamically manages two network services:
- **Keepalived**: VRRP-based virtual IP failover
- **Traefik**: L4 (TCP/UDP) and L7 load balancing

The agent receives configuration from the Kowabunga control plane (kahuna) and renders configuration templates for each managed service, then controls them via systemd.

## Commands

```bash
make all       # Full workflow: mod, fmt, vet, lint, build
make build     # Build binary to bin/konvey
make tests     # Run tests with coverage (coverage.txt)
make fmt       # Format with gofmt
make vet       # Run go vet
make lint      # Run golangci-lint
make vuln      # Check known vulnerabilities (govulncheck)
make sec       # Security checks (gosec)
make mod       # Download and tidy Go modules
make deb       # Build Debian package (Ubuntu Noble 24.04 LTS)
make apk       # Build Alpine Linux package
make update    # Update all dependencies
```

To run a single test:
```bash
go test ./... -count=1 -run TestName
```

## Architecture

The codebase is minimal — the heavy lifting lives in `github.com/kowabunga-cloud/common`.

```
cmd/konvey/main.go              # Entry point: calls konvey.Daemonize()
internal/konvey/konvey.go       # Defines konveyServices map + Daemonize()
internal/konvey/konvey_test.go  # Tests config template generation
```

**Core pattern**: `konveyServices` is a `map[string]*agents.ManagedService` that declares which systemd services to manage and which config templates to render into which paths. `Daemonize()` passes this map to `agents.KontrollerDaemon()` from the common package, which owns the daemon lifecycle, WebSocket control plane connection, and service management logic.

**Config templates** come from `github.com/kowabunga-cloud/common/agents/templates`:
- `KeepalivedConfTemplate("konvey")` → `/etc/keepalived/keepalived.conf`
- `TraefikConfTemplate("konvey")` → `/etc/traefik/traefik.yml`
- `TraefikLayer4ConfTemplate("konvey", "tcp")` → `/etc/traefik/conf.d/tcp.yml`
- `TraefikLayer4ConfTemplate("konvey", "udp")` → `/etc/traefik/conf.d/udp.yml`

When adding new managed services or config files, the pattern is: add an entry to `konveyServices` with the appropriate `agents.ManagedService` struct. Template logic and daemon plumbing are handled upstream in the common package.

## Tech Stack

- **Go 1.25.0**
- **kowabunga-cloud/common** — shared agent framework (templates, daemon lifecycle, systemd integration)
- **coreos/go-systemd/v22** — systemd unit management
- **gorilla/websocket** — control plane communication
- Target platforms: Ubuntu 24.04 LTS and Alpine Linux, amd64 and arm64
