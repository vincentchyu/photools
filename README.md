# 📸 photools

<p align="center">
  <img src="img/AppIcon.png" width="128" height="128" alt="photools App Icon" style="border-radius: 28px; box-shadow: 0 12px 30px rgba(0,0,0,0.35);" />
</p>

<p align="center">
  <b>High-Performance Automated Photo Processing, GPS Interpolation & Offline Geocoding Toolkit</b>
</p>

<p align="center">
  <a href="README.md"><b>English</b></a> | <a href="README_zh.md"><b>简体中文</b></a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-black?style=flat-square&logo=apple" alt="Platform" />
  <img src="https://img.shields.io/badge/Architecture-Modular%20Capability%20Pipeline-blue?style=flat-square" alt="Architecture" />
  <img src="https://img.shields.io/badge/Engine-ExifTool%20Stay--Open-brightgreen?style=flat-square" alt="ExifTool" />
  <img src="https://img.shields.io/badge/Spatial%20Index-3D%20KD--Tree-orange?style=flat-square" alt="KD-Tree" />
</p>

---

## 🌟 Overview

**photools** is an automated, high-performance photo processing pipeline engineered specifically for photographers (supporting Nikon RAW `.NEF`, Sony `.ARW`, Canon `.CR2/.CR3`, Fuji `.RAF`, Leica `.DNG`, Apple ProRAW, and high-quality JPGs).

It eliminates the tedious manual friction after importing photos to your computer:
1. **GPX Track Matching**: Matches camera timestamps against mobile GPX tracks with sub-second precision;
2. **GPS Intelligent Interpolation**: Recovers missing GPS coordinates via spherical great-circle time-weighted interpolation;
3. **Offline 3D KD-Tree Reverse Geocoding**: Embeds 5-tier Chinese administrative metadata (Country, Province, City, District, POI / Scenic Area) without internet;
4. **Atomic Date-Based Archiving**: Automatically cleans, renames (`YYYY-MM-DD-basename`), and archives paired assets into `Processed/YYYY/MMDD/`.

---

## 🖥️ macOS Native Application Showcase

photools features a high-performance native macOS desktop application built with SwiftUI and Go C-Shared FFI direct engine integration (`libphotools.dylib`, ~0.1ms latency). It includes **real-time Chinese & English language switching without restarting**, live EXIF photo inspection, and one-click snapshot backups.

### 1. Multi-Phase Pipeline Dashboard & Real-Time Console
> Multi-barrier pipeline execution dashboard providing granular controls for **GPX track matching**, **GPS intelligent interpolation**, **offline reverse geocoding**, and **date-based archiving** with a real-time event streaming console.

<p align="center">
  <img src="img/en/1-Pipeline-Dashboard.png" alt="Pipeline Dashboard & Real-Time Console" width="100%" />
</p>

---

### 2. Inbox Photos Management & Live EXIF Inspector
> High-performance photo gallery with multi-criteria filtering, paired RAW+JPG group synchronization, deep optical shooting parameters (Shutter, Aperture, ISO, Lens), and instant 3D KD-Tree reverse geocoding preview.

<p align="center">
  <img src="img/en/2-Inbox-Photos-Inspector.png" alt="Inbox Photos & Live EXIF Inspector" width="100%" />
</p>

---

### 3. Built-In Technical Documentation & Offline Geodata Tools
> **Left**: Comprehensive offline architecture specifications, memory rules, and user guides.<br/>
> **Right**: Built-in 3D KD-Tree offline geodata manager with continent dataset status and sub-millisecond coordinate lookup testing.

<p align="center">
  <img src="img/en/3-Guide-And-Geocoding.png" alt="User Guides & Offline Geodata Manager" width="100%" />
</p>

---

## ⚡ Core Capabilities (The 4 Pillars)

```
[Inbox RAW/JPG] ──▶ 1. GPX Matching ──▶ 2. GPS Interpolate ──▶ 3. Reverse Geocode ──▶ 4. Date Archive ──▶ [Processed/YYYY/MMDD/]
                       (P10 · Phase 1)     (P15 · Phase 2)        (P20 · Phase 3)       (P100 · Phase 4)
```

| Pillar / Plugin | Priority | Phase | Description |
| :--- | :---: | :---: | :--- |
| **`gpx_matching`** | `P10` | `Phase 1` | **GPX Track Matching**: Matches photo timestamps with GPX tracks, writes coordinates to Primary RAW, and synchronizes to companion JPG/XMP files. |
| **`gps_interpolate`** | `P15` | `Phase 2` | **Intelligent Time-Weighted Interpolation**: Recovers missing GPS coordinates via spherical great-circle interpolation or nearest-station inheritance using an $O(\log K)$ nanosecond `AnchorIndex`. |
| **`reverse_geocode`** | `P20` | `Phase 3` | **Offline 3D KD-Tree Geocoding**: Queries offline multi-continent datasets to tag Country, Province, City, District, and Scenic Spot POIs into IPTC/XMP tags. |
| **`date_archive`** | `P100` | `Phase 4` | **Date-Based Atomic Archiving**: Extracts `DateTimeOriginal`, standardizes filenames (`YYYY-MM-DD-basename`), and moves companion units into `Processed/YYYY/MMDD/`. |

---

## 🚀 Interactive Terminal TUI Workbench

For keyboard-driven workflows, photools includes a rich interactive TUI powered by Bubble Tea:

```bash
# Launch interactive TUI workbench
./photools tui
```

```text
╭────────────────────────────────────────────────────────────────────────────╮
│ ⚡ Capabilities Self-Check & Progressive Loading...                       │
│  [✔] Ready   GPX Track Matching & GPS Sync (ExifTool v12.76)               │
│  [✔] Ready   GPS Spherical Time-Weighted Interpolation (Window: 15m)       │
│  [⚙️] Loading Offline Reverse Geocoding (china.json · 927,314 points 62%)   │
│  [✔] Ready   Date-based Atomic Archive & Normalization                     │
╰────────────────────────────────────────────────────────────────────────────╯
```

- **`[1/2/3/4]` or `[Space]`**: Toggle specific capability plugins.
- **`[O]`**: Open plugin-specific self-describing configuration modal (e.g. adjust interpolation window, time offset).
- **`[S]`**: Global settings modal (Workspace path, Flat/In-Place mode, Worker concurrency, Soft-degrade policy) with `[Tab]` path completion.
- **`[Enter]`**: Trigger Dry-Run inspection, followed by immediate pipeline execution.

---

## 🛠️ CLI Usage & Scripting

### 1. Composite Pipeline (`pipeline`)
```bash
# Run full automated pipeline (GPX Matching + Geocoding + Date Archiving)
photools pipeline

# Enable GPS intelligent time-weighted interpolation (e.g., 1-hour search window)
photools pipeline --interpolate=true --interpolate-window=1h

# In-Place / Flat Mode (Process, geocode, and standardize directly in the source directory)
photools pipeline --flat=true --in-place=true

# Fault-tolerant soft degradation (Allow non-GPS photos to safely archive by date)
photools pipeline --allow-no-gps=true
```

### 2. Standalone Subcommands
```bash
# Geotag with time offset compensation (+5 seconds)
photools geotag -geosync +00:00:05 -workers 8

# Standalone offline reverse geocoding
photools geocode -dir /path/to/Inbox

# Standalone date-based organizer
photools organize-by-date -source-dir /path/to/source -target-dir /path/to/output

# Deep EXIF inspection (Exposure parameters, lens model, GPS, IPTC)
photools inspect /path/to/photo.NEF

---

## 📦 Build & Packaging Guide

### 1. Requirements
- **Go**: `1.21+`
- **Swift / Xcode Command Line Tools**: `Swift 5.9+` (for macOS Native App)
- **ExifTool**: `12.0+` (`brew install exiftool`)

---

### 2. Backend Core Components Build (Go C-Shared & CLI)

The backend is built in Go and provides two core targets:

```bash
# 1. Compile Go C-Shared dynamic library (for Swift GUI in-process FFI, output: dist/libphotools.dylib)
go build -buildmode=c-shared -o dist/libphotools.dylib ./cmd/photools-cshared

# 2. Compile standalone CLI & TUI binary (output: dist/photools)
go build -ldflags="-s -w" -o dist/photools ./cmd/photools
```

> [!NOTE]
> **Embedded Mapping Dictionaries (`//go:embed data`)**:
> Administrative division and EN-ZH mapping dictionaries under `pkg/geodata/data/` (`admin1CodesASCII_zh.json`, `admin2Codes_zh.json`, `country_codes.json`, etc.) are **automatically embedded into `libphotools.dylib` and the `photools` binary** at compile-time. When distributing the `.dylib` or `.dmg` to other users, these dictionaries are fully bundled and require no manual file copying. (To regenerate dictionaries, run `python3 pkg/geodata/data/generate_all.py`).

---

### 3. Offline High-Precision Geodata Installation (Required for Reverse Geocoding)

To enable 3D KD-Tree offline reverse geocoding (e.g. reverse lookup across 715k+ China POIs and administrative names), the corresponding regional database must be installed. **Because of their large file size, coordinates databases are not embedded into the dynamic library and must be installed explicitly in new environments**:

```bash
# 1. Check local geodata installation status
./dist/photools geodata status

# 2. Install China high-precision offline geodata (Recommended, saved to ~/.config/photools/geodata/)
./dist/photools geodata install china

# 3. (Optional) Install other continental datasets on demand
./dist/photools geodata install asia       # Asia
./dist/photools geodata install europe     # Europe
./dist/photools geodata install north-america # North America
```

---

### 4. macOS Native App Packaging (`.app` & `.dmg`)

photools provides an automated all-in-one packaging script [`script/build_and_run.sh`](script/build_and_run.sh) that compiles the Go C-Shared library (`libphotools.dylib`), builds the Swift GUI, embeds the CLI binary, packages the ExifTool engine runtime, configures `@rpath`, and performs local Ad-hoc code signing:

```bash
# 1. Build self-contained standalone App Bundle (Output: dist/photoolsApp.app)
./script/build_and_run.sh --build-only

# 2. Build and launch immediately for testing
./script/build_and_run.sh run

# 3. Create standard macOS distributable .dmg installer (for distribution)
hdiutil create -volname "photools" -srcfolder dist/photoolsApp.app -ov -format UDZO dist/photools-macOS.dmg
```

**App Bundle Output Architecture (`dist/photoolsApp.app`)**:
```text
photoolsApp.app/
└── Contents/
    ├── MacOS/
    │   ├── photoolsApp       # Native SwiftUI Executable
    │   └── photools            # Embedded Go CLI Engine
    ├── Frameworks/
    │   └── libphotools.dylib   # In-Process C-Shared FFI Dynamic Library (with embedded mapping dictionaries)
    ├── Resources/
    │   ├── docs/               # Technical Guides & Architecture Specs
    │   └── vendor/exiftool/    # Embedded ExifTool Runtime (Zero external dependency)
    └── Info.plist
```

> [!IMPORTANT]
> **Client Distribution Notes**:
> When distributed via `.dmg` to other users, `libphotools.dylib` will automatically resolve embedded name translations. If the recipient wants high-precision offline reverse geocoding, they need to run `photools geodata install china` once in their terminal, or have their `~/.config/photools/geodata/` directory pre-provisioned.

---

### 5. Standalone CLI / TUI Compilation & Distribution

To build stripped, high-performance standalone CLI binaries:

```bash
# Local stripped compilation
go build -ldflags="-s -w" -o photools ./cmd/photools

# Cross-platform release packaging
# macOS Apple Silicon (arm64)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/photools-darwin-arm64 ./cmd/photools

# macOS Intel (amd64)
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/photools-darwin-amd64 ./cmd/photools

# Linux x86_64 (amd64)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/photools-linux-amd64 ./cmd/photools
```

---

## 📊 Performance & Benchmark (Apple Silicon)

Benchmarked on **Apple M1 Max** (`arm64`):

```bash
go test -run=^$ -bench=. -benchmem ./internal/capabilities/gpsinterpolate/...
```

| Benchmark Target | Latency / op | Throughput | Memory Alloc |
| :--- | :---: | :---: | :---: |
| **`AnchorIndex` Binary Search (10,000 pts)** | **`205 ns/op`** | **`4,870,000 ops/sec`** | `0 B/op` |
| **`AnchorIndex` Batch Tree Build (1,000 assets)** | **`0.52 ms/op`** | **`1,920 batches/sec`** | `24 KB/op` |
| **`ExecuteProcess` Per-Photo Interpolation & Ingestion** | **`0.11 ms/op`** | **`8,880 photos/sec`** | `1.2 KB/op` |
| **`ExifTool` Stay-Open Daemon Pool** | **`1.2 ms/op`** | **`830 reads/sec`** | Zero-fork overhead |

---

## ⚙️ Configuration File (`plugins.json`)

photools automatically generates its persistent configuration at `~/.config/photools/plugins.json`:

```json
{
  "plugins": [
    {
      "id": "gpx_matching",
      "name": "GPX Track Matching",
      "priority": 10,
      "enabled": true
    },
    {
      "id": "gps_interpolate",
      "name": "GPS Intelligent Interpolation",
      "priority": 15,
      "enabled": true,
      "options": {
        "window": "15m"
      }
    },
    {
      "id": "reverse_geocode",
      "name": "Offline Reverse Geocoding",
      "priority": 20,
      "enabled": true
    },
    {
      "id": "date_archive",
      "name": "Date-Based Normalization Archive",
      "priority": 100,
      "enabled": true
    }
  ]
}
```

---

## 📖 Technical Documentation

- 📖 **[System Architecture & Phased Scheduler Design](docs/ARCHITECTURE_AND_DESIGN.md)**
- 🍏 **[macOS Native Client Architecture & C-Shared FFI Design](docs/MACOS_CLIENT_TECHNICAL_DESIGN.md)**
- ⚙️ **[Configuration Schema & Dynamic Settings Panel Design](docs/CONFIGURATION_AND_SETTINGS_DESIGN.md)**
- 🛡️ **[ExifTool Stay-Open Daemon Pool & Data Safety Design](docs/EXIFTOOL_IO_AND_SAFETY_DESIGN.md)**
- 🗺️ **[GeoNames Offline Data Pack & 3D KD-Tree Spatial Index Design](docs/GEONAMES_AND_GEOCODING_DESIGN.md)**

---

## 📄 License

photools is licensed under the [MIT License](LICENSE).
