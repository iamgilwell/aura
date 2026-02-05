# APO — AI-Powered Process Optimizer

**APO** is a Linux process management tool that combines real-time system monitoring with Anthropic Claude AI to intelligently identify and terminate wasteful processes. It features an htop-like terminal UI, automatic and interactive modes, power savings tracking, and a multi-layered safety system to protect critical services.

Built in Go. Powered by Claude.

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [Interactive TUI](#interactive-tui)
- [Configuration](#configuration)
- [AI Engine](#ai-engine)
- [Safety System](#safety-system)
- [Power Tracking](#power-tracking)
- [Process Classification](#process-classification)
- [Process Monitoring](#process-monitoring)
- [Daemon Mode](#daemon-mode)
- [Audit Trail](#audit-trail)
- [Testing](#testing)
- [Project Structure](#project-structure)
- [Dependencies](#dependencies)
- [License](#license)

---

## Features

- **Real-time Process Monitoring** — Scans `/proc` at configurable intervals, tracks CPU, memory, and I/O with trend analysis
- **AI-Powered Decisions** — Anthropic Claude evaluates processes and recommends terminate, keep, notify, or throttle actions
- **htop-like TUI** — Interactive terminal UI with sortable process table, AI decision panel, and keyboard controls
- **4-Level Safety System** — From fully automatic to monitor-only, with protected process lists and kernel thread detection
- **Power Savings Tracking** — Estimates watts saved per terminated process with monthly kWh/cost projections
- **Decision Caching** — LRU cache with TTL prevents redundant API calls for similar processes
- **Process Dependency Mapping** — Builds parent-child trees, identifies orphan risk, suggests safe termination order
- **Append-Only Audit Trail** — JSON audit log of every AI decision and termination event
- **Multiple Operating Modes** — Interactive TUI, text monitor, YOLO (fully automatic), and background daemon
- **Flexible Configuration** — YAML config file, environment variables, and CLI flags

---

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  /proc scan  │────▶│   Monitor    │────▶│   Classifier    │
│  (jiffies,   │     │  (CPU delta, │     │  (User/System/  │
│   status,    │     │   trends)    │     │   Kernel/Essen) │
│   io, cmd)   │     └──────┬───────┘     └────────┬────────┘
└──────────────┘            │                      │
                            ▼                      ▼
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  Anthropic   │◀───│  AI Engine   │◀────│  Safety Manager │
│  Claude API  │───▶│  (+ Cache)   │────▶│  (Protection +  │
│              │     │              │     │   Consent)      │
└──────────────┘     └──────┬───────┘     └─────────────────┘
                            │
                 ┌──────────┼──────────┐
                 ▼          ▼          ▼
          ┌──────────┐ ┌────────┐ ┌──────────┐
          │  Process  │ │ Power  │ │  Audit   │
          │  Manager  │ │ Tracker│ │  Trail   │
          │ (SIGTERM/ │ │ (watts,│ │  (JSON   │
          │  SIGKILL) │ │  kWh)  │ │   log)   │
          └──────────┘ └────────┘ └──────────┘
```

---

## Installation

### Prerequisites

- Go 1.21+
- Linux (reads from `/proc`)
- Anthropic API key (for AI features)

### Build from Source

```bash
git clone https://github.com/iamgilwell/aura.git
cd aura
go build -o apo .
```

### Verify Installation

```bash
./apo --help
./apo status
```

---

## Quick Start

```bash
# 1. Copy and edit configuration
cp config.example.yaml config.yaml
# Set your ANTHROPIC_API_KEY in config.yaml or environment

# 2. Monitor processes (text mode)
./apo monitor

# 3. Launch interactive TUI
./apo interactive

# 4. YOLO mode — AI auto-terminates wasteful processes
export ANTHROPIC_API_KEY=sk-ant-...
sudo ./apo yolo
```

---

## Commands

### `apo monitor`

Foreground process monitoring with text output. Displays a refreshing table of the top 30 processes sorted by CPU usage.

```bash
./apo monitor
./apo monitor --verbose
```

### `apo interactive`

Launches the full htop-like terminal UI with process table, AI decision panel, dashboard, and keyboard controls.

```bash
./apo interactive
./apo interactive --config /path/to/config.yaml
```

### `apo yolo`

AI-driven automatic mode. Sets consent level to 0 (fully automatic) and lets Claude evaluate and terminate wasteful user processes without confirmation.

**Requires `ANTHROPIC_API_KEY`.**

```bash
sudo ./apo yolo
```

### `apo status`

Displays current system state, daemon status, and configuration summary.

```bash
./apo status
```

Example output:

```
╔══════════════════════════════════════════╗
║     APO - AI-Powered Process Optimizer   ║
╚══════════════════════════════════════════╝

Daemon:     Not running

System Metrics:
  CPU Usage:    12.3%
  Memory:       70.7% (16930 MB / 23955 MB)
  Load Average: 2.46, 2.04, 2.36
  Uptime:       11h24m26.12s

Configuration:
  AI Enabled:     true
  AI Model:       claude-sonnet-4-5-20250929
  Consent Level:  2 (Confirm All)
  Scan Interval:  2s
  Aggressiveness: 5/10
  API Key Set:    false

Protected Processes: 15 configured
Never Terminate:     4 configured
```

### `apo start`

Starts APO in background daemon mode.

```bash
./apo start --daemon          # Background daemon
./apo start                   # Foreground (same as monitor)
```

PID file stored at `/tmp/apo.pid`.

### `apo stop`

Stops the background daemon by sending SIGTERM.

```bash
./apo stop
```

### `apo logs`

View or follow the APO log file.

```bash
./apo logs                    # Print entire log
./apo logs --follow           # Tail -f style
```

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `./config.yaml` | Path to config file |
| `--verbose` | bool | `false` | Enable verbose/debug output |

---

## Interactive TUI

The interactive mode provides an htop-like terminal interface with three panels:

```
┌─ APO - AI-Powered Process Optimizer ──────────────────────────────────┐
│ Runtime: 5m32s │ Mode: Interactive │ Safety: Confirm All (L2) │ AI: ON│
├─ Processes ───────────────────────────────────────────────────────────┤
│ PID    NAME          USER    CPU%   MEM%  MEM(MB)  IO    CAT    CMD  │
│ 1234   firefox       user    45.2   12.3  2940.5   1.2G  User   ... │
│ 5678   chrome        user    32.1    8.7  2084.2   890M  User   ... │
│ ...                                                                   │
├─ AI Decisions ────────────────────────────────────────────────────────┤
│ 14:32:05 keep       PID=1234  firefox    conf=0.85 risk=0.20 ...    │
│ 14:32:05 terminate  PID=9999  zombie-app conf=0.92 risk=0.10 ...    │
├───────────────────────────────────────────────────────────────────────┤
│ F1:AI History F2:Suggestions F3:Power F5:Refresh F6:Sort F9:Kill F10:│
└───────────────────────────────────────────────────────────────────────┘
```

### Keybindings

| Key | Action |
|-----|--------|
| **F1** | Show AI decision history |
| **F2** | Toggle AI suggestions on/off |
| **F3** | Show power savings metrics |
| **F4** | Show process dependency tree for selected process |
| **F5** | Refresh display |
| **F6** | Cycle sort field (CPU → Memory → PID → Name → IO) |
| **F7** | Decrease AI aggressiveness (min 1) |
| **F8** | Increase AI aggressiveness (max 10) |
| **F9** | Terminate selected process (with safety checks) |
| **F10** | Quit |
| **q/Q** | Quit |
| **a/A** | AI-evaluate selected process |

### Sort Fields

Cycle through with F6:

1. **CPU%** (default, descending)
2. **MEM%**
3. **PID**
4. **NAME**
5. **IO** (read + write bytes)

### Process Table Columns

| Column | Description |
|--------|-------------|
| PID | Process ID |
| NAME | Process name (color-coded by category) |
| USER | Owner username |
| CPU% | CPU usage percentage (red >50%, yellow >20%) |
| MEM% | Memory usage percentage (red >50%, yellow >20%) |
| MEM(MB) | Resident memory in megabytes |
| IO(R+W) | Total I/O bytes (formatted as B/K/M/G) |
| CAT | Category (User/System/Kernel/Essential) |
| STATE | Process state |
| COMMAND | Full command line |

### Color Coding

| Color | Meaning |
|-------|---------|
| Green | User processes |
| Blue | System processes |
| Gray | Kernel threads |
| Red | Essential/protected processes |

---

## Configuration

APO searches for configuration in this order:

1. Path specified via `--config` flag
2. `./config.yaml` (current directory)
3. `~/.apo/config.yaml` (user home)
4. `/etc/apo/config.yaml` (system-wide)

Environment variables override config file values with the `APO_` prefix (e.g., `APO_MONITORING_SCAN_INTERVAL`). The `ANTHROPIC_API_KEY` environment variable is read directly.

### Full Configuration Reference

```yaml
# Anthropic Claude API settings
anthropic:
  api_key: ""                          # API key (or set ANTHROPIC_API_KEY env var)
  model: "claude-sonnet-4-5-20250929"  # Claude model to use

# Process monitoring settings
monitoring:
  scan_interval: "2s"        # How often to scan /proc
  cpu_threshold: 80.0        # CPU% threshold for AI evaluation
  memory_threshold: 80.0     # Memory% threshold for AI evaluation
  io_threshold: 104857600    # I/O bytes/s threshold (100MB/s)
  history_size: 100          # Number of historical snapshots to retain

# AI decision engine settings
ai:
  enabled: true              # Enable/disable AI evaluation
  confidence_threshold: 0.7  # Minimum confidence to act on decisions (0.0-1.0)
  cache_size: 500            # Maximum cached AI decisions
  cache_ttl: "30m"           # Cache entry time-to-live
  max_requests_per_min: 30   # Rate limit for API calls
  aggressiveness: 5          # 1 (conservative) to 10 (aggressive)

# Safety system settings
safety:
  consent_level: 2           # 0=automatic, 1=notify system, 2=confirm all, 3=monitor only
  protected_processes:       # Processes that require elevated consent
    - systemd
    - init
    - sshd
    - dbus-daemon
    - NetworkManager
    - pulseaudio
    - pipewire
    - Xorg
    - Xwayland
    - gnome-shell
    - kwin
    - sway
    - gdm
    - lightdm
    - login
  never_terminate:           # Processes that can NEVER be terminated
    - systemd
    - init
    - kernel
    - kthreadd
  terminate_timeout: "5s"    # Time between SIGTERM and SIGKILL

# Notification and logging settings
notifications:
  log_file: "apo.log"           # Application log file
  audit_file: "apo-audit.log"   # AI decision audit trail
  verbose: false                 # Enable debug logging
  color_enabled: true            # Color-coded terminal output

# Power estimation settings
power:
  cpu_watt_per_percent: 0.5  # Watts per 1% CPU usage
  memory_watt_per_mb: 0.001  # Watts per MB of resident memory
  disk_watt_per_mbps: 0.02   # Watts per MB/s of disk I/O
  track_savings: true        # Enable power savings tracking
```

---

## AI Engine

### How It Works

1. **Process Selection** — The monitor identifies processes exceeding CPU/memory thresholds
2. **Cache Check** — A process signature (name + user + resource bucket) is checked against the LRU cache
3. **API Call** — If not cached, a structured prompt is sent to Claude with full process context and system state
4. **Response Parsing** — Claude returns a JSON decision with action, confidence, risk score, and estimated power savings
5. **Safety Validation** — The decision passes through the safety manager before execution
6. **Caching** — The decision is cached to avoid redundant API calls for similar processes

### AI Actions

| Action | Description |
|--------|-------------|
| `terminate` | Process should be killed to save resources |
| `keep` | Process should continue running |
| `notify` | Process is suspicious but doesn't warrant termination yet |
| `throttle` | Process should have its resources limited |

### Decision Response Format

```json
{
  "action": "terminate",
  "confidence": 0.92,
  "reason": "Zombie process consuming CPU with no parent — safe to terminate",
  "risk_score": 0.1,
  "savings_watt": 2.5
}
```

### Aggressiveness Levels

The aggressiveness setting (1-10) is passed directly to the AI system prompt and affects how Claude evaluates processes:

| Level | Behavior |
|-------|----------|
| 1-3 | Conservative — only terminate clearly wasteful/zombie processes |
| 4-6 | Moderate — terminate idle background processes with high resource usage |
| 7-10 | Aggressive — terminate anything not deemed essential |

Adjustable at runtime via F7/F8 in the interactive TUI.

### Caching Strategy

- CPU and memory values are bucketed to the nearest 5% to avoid cache misses from minor fluctuations
- Cache key: SHA-256 hash of `{name}|{user}|{cpuBucket}|{memBucket}|{category}`
- Default cache size: 500 entries, 30-minute TTL
- LRU eviction when full
- Cached responses are flagged with `FromCache: true`

### Fallback Behavior

When the API is unavailable (network error, rate limit, invalid key), the engine returns a safe default:

- Action: `keep`
- Confidence: `0.0`
- Reason: `"AI unavailable — defaulting to keep"`

The engine never blocks or crashes due to API failures.

---

## Safety System

APO implements multiple layers of protection to prevent terminating critical processes.

### Consent Levels

| Level | Name | Behavior |
|-------|------|----------|
| **0** | Fully Automatic | Terminate without any confirmation |
| **1** | Notify for System | Auto-terminate user processes; require confirmation for system/essential |
| **2** | Confirm All | Require confirmation for every termination *(default)* |
| **3** | Monitor Only | Never terminate anything; observation mode only |

### Protection Layers (checked in order)

1. **PID Protection** — PID 1 (init) and PID 2 (kthreadd) are always protected
2. **Kernel Thread Detection** — Any process with `CategoryKernel` is always protected
3. **Never-Terminate List** — Hardcoded list: `systemd`, `init`, `kernel`, `kthreadd`
4. **Protected Process List** — Configurable list of essential services (sshd, dbus-daemon, display managers, etc.)

### Termination Flow

```
User/AI requests termination
        │
        ▼
  IsProtected(proc)?  ──yes──▶  BLOCKED (with reason)
        │ no
        ▼
  ValidateTermination(proc)?  ──fail──▶  BLOCKED (with reason)
        │ pass
        ▼
  NeedsConfirmation(proc)?  ──yes──▶  Prompt user
        │ no                              │
        ▼                                 ▼
  Send SIGTERM  ◀─────────────────  User confirms
        │
        ▼
  Wait terminate_timeout (5s)
        │
        ▼
  Process still alive?  ──yes──▶  Send SIGKILL
        │ no
        ▼
  Log savings + audit
```

---

## Power Tracking

### Estimation Formula

```
Process Power (watts) = (CPU% × 0.5) + (MemoryMB × 0.001) + (IO_MBps × 0.02)
```

| Component | Default Coefficient | Example |
|-----------|-------------------|---------|
| CPU | 0.5 W per 1% CPU | 50% CPU = 25.0 W |
| Memory | 0.001 W per MB | 2 GB = 2.048 W |
| Disk I/O | 0.02 W per MB/s | 10 MB/s = 0.2 W |

### Projections

- **Monthly kWh**: `watts × 24 hours × 30 days / 1000`
- **Monthly Cost**: `kWh × rate_per_kWh`
- **Example**: Saving 10W continuously = 7.2 kWh/month = ~$0.86/month at $0.12/kWh

### Metrics Dashboard (F3)

```
Power Metrics
─────────────────────────────────────
Total Saved:        12.50 W
Session Duration:   2h15m30s
Monthly Projection: 6.48 kWh
Events Recorded:    8

Recent Savings:
  14:32:05  zombie-proc    PID=9999   +2.50W  AI termination
  14:28:12  leaked-worker  PID=8888   +5.00W  Manual termination
```

---

## Process Classification

APO categorizes every process into one of four categories:

| Category | Color (TUI) | Detection Rules |
|----------|-------------|-----------------|
| **Kernel** | Gray | PID ≤ 2, PPID = 2 (kthreadd child), or bracketed cmdline `[kworker/0:0]` |
| **Essential** | Red | Name appears in `protected_processes` config list |
| **System** | Blue | UID = 0 (root) or name in known system daemons list |
| **User** | Green | Everything else |

### Known System Daemons (auto-classified)

`systemd`, `init`, `rsyslogd`, `syslogd`, `journald`, `udevd`, `dbus-daemon`, `polkitd`, `accounts-daemon`, `cron`, `atd`, `acpid`, `thermald`, `irqbalance`, `snapd`, `packagekitd`, `udisksd`, `colord`, `cupsd`, `avahi-daemon`, `bluetoothd`, `wpa_supplicant`, `dhclient`, `NetworkManager`, `ModemManager`

---

## Process Monitoring

### Data Sources

APO reads directly from the Linux `/proc` filesystem:

| File | Data Extracted |
|------|----------------|
| `/proc/[pid]/stat` | PID, name, state, PPID, CPU jiffies (utime/stime), start time |
| `/proc/[pid]/status` | UID, VmRSS (resident memory) |
| `/proc/[pid]/cmdline` | Full command line |
| `/proc/[pid]/io` | Read/write bytes |
| `/proc/meminfo` | Total memory, available memory |
| `/proc/loadavg` | 1/5/15 minute load averages |
| `/proc/uptime` | System uptime |
| `/proc/stat` | Boot time (for calculating process start time) |

### CPU Calculation

CPU usage is calculated from jiffies deltas between scans:

```
CPU% = ((utime_delta + stime_delta) / clock_ticks) / elapsed_seconds × 100
```

Where `clock_ticks` = 100 (standard Linux `_SC_CLK_TCK`).

### Trend Analysis

Each scan calculates deltas from the previous scan:

- **CPU Trend**: Current CPU% − Previous CPU% (positive = increasing)
- **Memory Trend**: Current Mem% − Previous Mem% (positive = growing)

---

## Daemon Mode

### Start

```bash
./apo start --daemon
```

- Forks a background `apo monitor` process
- Writes PID to `/tmp/apo.pid`
- Detaches from terminal

### Stop

```bash
./apo stop
```

- Reads PID from `/tmp/apo.pid`
- Sends SIGTERM to the daemon
- Removes PID file

### Check Status

```bash
./apo status
```

Shows whether the daemon is running and its PID.

---

## Audit Trail

APO maintains an append-only JSON audit log at the configured `audit_file` path (default: `apo-audit.log`). Every AI decision and termination is recorded:

```json
{"timestamp":"2025-01-15T14:32:05Z","event":"ai_decision","decision":{"process_pid":9999,"process_name":"zombie-app","action":"terminate","confidence":0.92,"reason":"Zombie process","risk_score":0.1,"savings_watt":2.5}}
{"timestamp":"2025-01-15T14:32:06Z","event":"termination","details":"pid=9999 name=zombie-app reason=AI recommendation"}
```

### Event Types

| Event | Description |
|-------|-------------|
| `ai_decision` | AI evaluated a process (includes full decision response) |
| `termination` | A process was terminated (includes PID, name, reason) |
| `yolo_start` | YOLO mode was activated |
| `yolo_stop` | YOLO mode was deactivated (includes total power saved) |

---

## Testing

Run all tests:

```bash
go test ./tests/... -v
```

### Test Coverage

| Test File | Tests | What's Covered |
|-----------|-------|----------------|
| `ai_test.go` | 4 | Process signature generation, LRU cache (put/get/eviction), TTL expiration, action constants |
| `monitor_test.go` | 4 | System metrics from /proc, monitor creation, category strings, process classification (5 subtests) |
| `safety_test.go` | 4 | Protected process detection (6 subtests), termination validation, consent level descriptions, confirmation logic per level |
| `power_test.go` | 4 | Power calculation with coefficients, metrics tracking, monthly kWh conversion, cost estimation |

**Total: 15 tests, all passing.**

---

## Project Structure

```
aura/
├── main.go                           # Entry point
├── go.mod / go.sum                   # Go module definition
├── config.example.yaml               # Example configuration
├── cmd/
│   ├── root.go                       # Root cobra command, config init
│   ├── monitor.go                    # apo monitor — text process viewer
│   ├── yolo.go                       # apo yolo — automatic AI mode
│   ├── interactive.go                # apo interactive — TUI launcher
│   ├── status.go                     # apo status — system status
│   ├── start.go                      # apo start --daemon
│   ├── stop.go                       # apo stop
│   └── logs.go                       # apo logs --follow
├── internal/
│   ├── config/
│   │   └── config.go                 # Viper config loading, struct defs
│   ├── monitor/
│   │   ├── process.go                # ProcessInfo, /proc parsing, SystemMetrics
│   │   ├── classifier.go             # Process categorization logic
│   │   └── monitor.go                # ProcessMonitor scan loop
│   ├── ai/
│   │   ├── types.go                  # DecisionRequest/Response, Action enum
│   │   ├── cache.go                  # LRU decision cache with TTL
│   │   └── engine.go                 # Claude API integration, prompt building
│   ├── safety/
│   │   ├── safety.go                 # Protection checks, termination validation
│   │   └── consent.go                # Consent levels 0-3
│   ├── power/
│   │   ├── calculator.go             # Power estimation formulas
│   │   └── metrics.go                # Savings tracking, projections
│   ├── process/
│   │   ├── manager.go                # SIGTERM/SIGKILL termination
│   │   └── dependencies.go           # Process tree, orphan detection
│   ├── notification/
│   │   ├── notifier.go               # Color terminal output, file logging
│   │   └── audit.go                  # Append-only JSON audit trail
│   └── ui/
│       ├── app.go                    # Main tview application, layout
│       ├── dashboard.go              # Top status bar
│       ├── processtable.go           # Sortable process table
│       ├── decisionpanel.go          # AI decision history panel
│       └── keybindings.go            # F1-F10 key handlers
├── configs/
│   └── config.example.yaml           # Example configuration
└── tests/
    ├── ai_test.go                    # Cache, signature tests
    ├── monitor_test.go               # /proc parsing, classification tests
    ├── safety_test.go                # Protection, consent tests
    └── power_test.go                 # Power calculation tests
```

---

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/spf13/cobra` | v1.10.2 | CLI command framework |
| `github.com/spf13/viper` | v1.21.0 | Configuration management |
| `github.com/rivo/tview` | v0.42.0 | Terminal UI framework |
| `github.com/gdamore/tcell/v2` | v2.13.8 | Terminal cell library |
| `github.com/anthropics/anthropic-sdk-go` | v1.21.0 | Anthropic Claude API client |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parsing |

---

## License

MIT
