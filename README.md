# makakito

<p align="center">
  <img src="assets/logo.png" alt="makakito" width="200" />
  <br/>
  <em>Chaos Engineering for teams that don't have a chaos engineering team.</em>
</p>

`makakito` is a **single binary** you run from the command line. It reads a YAML experiment file, injects failures into your infrastructure, checks your system's health before and after, and rolls everything back when done. No daemon, no central server, no Kubernetes operator — just a program you run and watch.

---

## How it works

When you run `chaos run experiment.yaml`, the binary:

1. **Validates** the experiment file against the schema.
2. **Checks your steady-state hypothesis** — runs probes (e.g. an HTTP health check) to confirm the system is healthy before anything is touched.
3. **Resolves targets** — finds the Docker containers, processes, or other resources that match your selector.
4. **Applies faults** — directly calls the relevant system or API: Docker API for container faults, Linux `tc-netem` for network faults, `os.Signal` for process kills, `runtime` goroutines for CPU/memory stress.
5. **Waits** for the fault duration you specified.
6. **Rolls back** — restores every resource that was changed, in reverse order, regardless of whether the experiment passed or failed. Rollback always runs, even on crash.
7. **Checks the hypothesis again** and reports pass or fail.
8. **Saves the result** to a local SQLite database (`~/.makakito/history.db`).

Nothing runs in the background after the command exits. There is no agent to install on your servers. The binary talks directly to the local Docker socket, the Linux kernel's traffic control subsystem, or the Kubernetes API — depending on what your experiment targets.

### What makakito is NOT

- Not a daemon or a service. It runs, does its job, and exits.
- Not a container. The binary runs natively on your host; it is not wrapped in Docker.
- Not a cloud platform. No data leaves your machine by default.
- Not a library. You consume it as a CLI tool, not as a Go package.

### Prerequisites by fault type

| Fault | What it needs |
|---|---|
| `container-stop`, `container-pause` | Docker daemon running, socket at `/var/run/docker.sock` |
| `cpu`, `memory` | Nothing — uses Go goroutines and `make([]byte, n)` |
| `process-kill` | Permission to send signals to the target PID |
| `network-latency`, `packet-loss` | Linux only; `CAP_NET_ADMIN` or root (uses `tc-netem` via netlink) |

---

## Installation

### Linux (recommended)

**Download the binary directly:**

```bash
# amd64
curl -L https://github.com/CyberArgonaut/makakito/releases/latest/download/chaos_linux_amd64.tar.gz \
  | tar xz -C /usr/local/bin/ chaos
chaos version
```

```bash
# arm64 (Raspberry Pi, AWS Graviton, etc.)
curl -L https://github.com/CyberArgonaut/makakito/releases/latest/download/chaos_linux_arm64.tar.gz \
  | tar xz -C /usr/local/bin/ chaos
chaos version
```

**Build from source** (requires Go 1.26+):

```bash
git clone https://github.com/CyberArgonaut/makakito
cd makakito
make build
sudo cp bin/chaos /usr/local/bin/
chaos version
```

### macOS

```bash
# Homebrew
brew install makakito

# or download directly
curl -L https://github.com/CyberArgonaut/makakito/releases/latest/download/chaos_darwin_arm64.tar.gz \
  | tar xz -C /usr/local/bin/ chaos
```

### Verify the install

```bash
chaos version
# makakito v0.1.0 (commit: abc1234, built: 2026-04-01T00:00:00Z)

chaos --help
```

---

## Quick Start

```bash
# 1. Generate a starter experiment for your stack
chaos init

# 2. Check the generated file looks right
chaos validate my-first-chaos-experiment.yaml

# 3. Simulate without touching anything real
chaos run --dry-run my-first-chaos-experiment.yaml

# 4. Run it for real
chaos run my-first-chaos-experiment.yaml

# 5. Review what happened
chaos history
```

---

## Your First Experiment

An experiment file has three parts: a **hypothesis** (what does healthy look like?), a **method** (what failure to inject), and **controls** (safety limits).

```yaml
# experiments/redis-blackout.yaml
version: "1"
name: Redis blackout
description: Validates that the API degrades gracefully when Redis is unavailable.
labels:
  phase: "1"
  team: platform
  env: staging

hypothesis:
  title: API returns 200 with cache-miss fallback during Redis outage
  probes:
    - type: http
      url: http://staging.myapp.internal/healthz
      expected_status: 200
      timeout: 5s

method:
  - type: fault
    name: Stop Redis container
    target:
      kind: docker
      selector:
        name: redis       # matches any container whose name contains "redis"
    fault:
      kind: container-stop
    duration: 60s
    rollback: auto        # chaos restarts the container when done

controls:
  blast_radius:
    max_targets: 1        # never stop more than 1 container, even if multiple match
  timeout: 3m             # hard kill-switch — experiment aborts after 3 minutes regardless
  environments:
    - staging             # chaos refuses to run if --environment is not "staging"
```

```bash
chaos run --environment staging experiments/redis-blackout.yaml

# time=... level=INFO msg="checking hypothesis (before)" experiment="Redis blackout"
# time=... level=INFO msg="probe result" type=http phase=before success=true
# time=... level=INFO msg="applying fault" action="Stop Redis container" resource=redis
# time=... level=INFO msg="fault active, waiting" duration=1m0s
# time=... level=INFO msg="checking hypothesis (after)" experiment="Redis blackout"
# time=... level=INFO msg="probe result" type=http phase=after success=true
#
# ✅ Experiment "Redis blackout" passed in 1m 2s
#    ID: exp-20260401-120000-042381
```

### What happened, step by step

1. chaos called `GET http://staging.myapp.internal/healthz` → got 200. Hypothesis confirmed.
2. chaos called `POST /containers/redis/stop` on the local Docker socket.
3. chaos waited 60 seconds.
4. chaos called `GET http://staging.myapp.internal/healthz` again → got 200. Hypothesis still holds.
5. chaos called `POST /containers/redis/start` on the Docker socket (auto rollback).
6. chaos wrote the result to `~/.makakito/history.db`.

---

## Dry Run

`--dry-run` goes through the full experiment flow but skips the actual fault injection. It will:
- Parse and validate the YAML.
- Resolve targets (connect to Docker, find matching containers).
- Run probes.
- Log what it *would* do instead of doing it.
- Roll back nothing (nothing was applied).

```bash
chaos run --dry-run experiments/redis-blackout.yaml
# [dry-run] Simulating experiment: Redis blackout
# ... probes run for real, faults are skipped ...
```

This is useful for validating that selectors match the right resources before running anything destructive.

---

## Safety Controls

Every experiment has hard limits that cannot be overridden at runtime (only `--force` bypasses `require_approval`, and that is logged):

```yaml
controls:
  blast_radius:
    max_targets: 2        # Hard cap. If selector matches 10 containers, only 2 are affected.
    percentage: 10        # Or: affect at most 10% of matched resources. Most restrictive wins.
  timeout: 5m             # The experiment is forcibly aborted after this. Default: 5 minutes.
  environments:
    - staging             # Requires --environment=staging flag. Prevents accidental prod runs.
  require_approval: true  # Pauses and asks for y/N before applying any fault.
```

Rollback always runs — even on timeout, panic, or SIGINT. The rollback stack is LIFO and uses a fresh context so a cancelled experiment context does not prevent cleanup.

---

## Fault Library (v0.1)

**Available now:**

| Fault | Kind | Target | Notes |
|---|---|---|---|
| Stop container | `container-stop` | Docker | Restarts on rollback |
| Pause container | `container-pause` | Docker | Unpauses on rollback |
| CPU stress | `cpu` | Any | Params: `cores` (default: all CPUs) |
| Memory pressure | `memory` | Any | Params: `size` (e.g. `100MB`, `1GB`) |
| Process kill | `process-kill` | Any process by PID | Params: `signal` (default: `SIGTERM`) |
| Network latency | `network-latency` | Linux host | Params: `interface`, `latency`, `jitter`. Requires `CAP_NET_ADMIN`. |
| Packet loss | `packet-loss` | Linux host | Params: `interface`, `percent`. Requires `CAP_NET_ADMIN`. |

**Coming in v0.2–v0.3:** Kubernetes pod kill, node drain, DNS failure, EC2 stop, clock skew, and more.

---

## Steady-State Probes (v0.1)

Probes run before and after the experiment. A failed before-probe aborts the experiment without applying any fault. A failed after-probe marks the experiment as failed (and rollback still runs).

```yaml
hypothesis:
  probes:
    - type: http
      url: http://api.internal/healthz
      expected_status: 200
      timeout: 5s
```

**Available now:** `http` — checks a URL for a specific status code.

**Coming in v0.2–v0.3:** `prometheus` (PromQL expression must evaluate to true), `log` (scan container logs for a pattern).

---

## CLI Reference

```
chaos version               Print version, commit, and build date
chaos init                  Wizard: ask a few questions, write a starter experiment YAML
chaos validate <file>       Parse and validate the YAML — print errors, exit non-zero on failure
chaos run <file>            Run an experiment
  --dry-run                 Simulate without applying faults (probes still run)
  --environment <env>       Assert this environment matches the allowlist
  --force                   Bypass require_approval (logs a warning)
  --json                    Print result as JSON
chaos rollback <id>         Show experiment details (full rollback from CLI coming in v0.2)
chaos status                List currently running experiments
chaos history               List past results
  --limit N                 How many to show (default: 20)
  --status <status>         Filter: passed, failed, aborted, error
  --json                    Print as JSON
chaos dashboard             (v0.2) Launch embedded web UI at localhost:7070
```

---

## Configuration

`makakito` looks for a config file at `./makakito.yaml` first, then `~/.makakito.yaml`. All keys can also be set via environment variables prefixed with `CHAOS_` (e.g. `CHAOS_STORE_PATH`).

```yaml
# makakito.yaml
log_level: info           # debug | info | warn | error
store:
  path: ~/.makakito/history.db
dashboard:
  port: 7070              # v0.2
kubernetes:               # v0.2
  kubeconfig: ~/.kube/config
  namespace: default
aws:                      # v0.3
  region: us-east-1
  profile: default
```

---

## Infrastructure Targets

| Target | Available |
|---|---|
| Docker / Docker Compose | v0.1 |
| Kubernetes | v0.2 |
| AWS (EC2, S3, RDS, Lambda) | v0.3 |
| SSH / bare metal | v0.3 |

---

## Roadmap

| Version | What ships |
|---|---|
| v0.1 | Single binary, YAML schema, Docker target, Phase 1 faults, SQLite history, full CLI |
| v0.2 | Embedded web dashboard, Kubernetes target, Slack/webhook notifications |
| v0.3 | Phase 2 faults (pod kill, DNS, EC2, clock skew), AWS target, Prometheus probes |
| v1.0 | Stable schema, documented plugin interface, full Phase 1+2 |
| v1.x | Phase 3 faults, eBPF network driver (no root required), multi-team approval |

---

## Contributing

```bash
git clone https://github.com/CyberArgonaut/makakito
cd makakito
make dev-setup    # downloads deps, installs golangci-lint
make test         # unit tests (no Docker required)
make test-integration  # needs Docker running
make build        # produces bin/chaos
```

**Adding a fault:**
1. Implement `engine.Fault` in `internal/fault/yourfault.go` — `Apply()` must always return a non-nil `RollbackFn`.
2. Register it in `internal/fault/registry.go`.
3. Add a YAML example to `experiments/`.
4. Add unit tests in `internal/fault/fault_test.go` with a mock target — the key invariant to test is that `rollback != nil` even when `Apply()` returns an error.

---

## License

MIT — see [LICENSE](./LICENSE).

---

> Built for the team that's responsible for reliability but hasn't had time to make chaos a discipline yet.
