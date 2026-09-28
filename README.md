# AirShop

A complete open-source e-commerce platform built in Go, on top of the
[Airway](https://github.com/daqing/airway) framework.
For the Chinese edition, see [README.zh-CN.md](README.zh-CN.md).

## About

AirShop implements a complete e-commerce system in Go — server-rendered,
database-backed, and packageable as a native desktop application. It is
under active development.

The current scope is the classic commerce essentials; AI-powered features
will follow in a third phase (yet to be planned).

## Roadmap

- **Phase 1 — Core commerce** *(current)*: implement the complete core
  commerce feature set.
- **Phase 2 — Beyond core**: once the core system is complete, development
  shifts to AirShop-specific features.
- **Phase 3 — AI features** *(to be planned)*: AI-powered capabilities will
  be provided in this phase.

## Feature scope

Storefront:

- Homepage
- Product catalog and details
- Shopping cart
- Orders and checkout
- Payments
- Coupons
- Account center
- Sign-up and login via phone number
- Address book
- Shipment tracking

Admin:

- Admin panel for managing the store

## Tech stack

- **Go** with the [Airway](https://github.com/daqing/airway) framework — a
  server-rendered web app with a CLI for migrations, scaffolding and a
  project REPL.
- **Databases**: PostgreSQL, MySQL or SQLite; optional Redis.
- **File storage**: local disk, Amazon S3, Cloudflare R2 or Tencent COS.
- **Desktop**: the same app packages as a native macOS / Windows / Linux
  application via Wails v3.

## Getting started

`.env` is created for you at scaffold time — open it and set `AIRWAY_ENV`
(e.g. `local`), a `DSN` and the `LISTEN` address (`host:port`, e.g. `:1900`):

```bash
airway db:create
airway db:migrate
go run .               # start the HTTP server (or: airway server)
```

## Common commands

```bash
airway generate api admin          # scaffold an API namespace
airway generate model post         # scaffold a model
airway generate migration create_posts
airway db:migrate
go run . repl                      # REPL with this project's models
```

## Desktop apps (macOS / Windows / Linux)

This project can be packaged as a native desktop application: the same web
stack runs on a local port inside the desktop process and a native WebView
window loads it, so server-rendered pages, cookie sessions, redirects and
WebSockets behave exactly as on the web — no application code changes.

### 1. Generate the desktop target

```bash
airway desktop:init
```

This creates a `desktop/` directory containing the Wails v3 project: the
window bootstrap, your `db/migrate` SQL embedded for automatic first-run
migrations, the plugin mirror, and build assets for all three platforms. It
also pins `github.com/wailsapp/wails/v3` in `go.mod`. Re-running the command
is safe: it re-syncs migrations and plugins but never touches
`desktop/main.go` (use `--force` to regenerate it).

### 2. Install the desktop toolchain (once)

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.24
go install github.com/go-task/task/v3/cmd/task@latest
```

Build requirements per platform:

| Platform | Requirement |
|---|---|
| macOS 12+ | Xcode command line tools (`xcode-select --install`) |
| Windows 10/11 | nothing extra to build; installers bundle the WebView2 bootstrapper |
| Linux | GTK4 + WebKitGTK 6.0 dev packages (Ubuntu 24.04+ / Debian 13+), e.g. `sudo apt install libgtk-4-dev libwebkitgtk-6.0-dev` |

### 3. Run in development

```bash
cd desktop
wails3 task dev
```

### 4. Build the installers

All artifacts land in `desktop/bin/`.

#### macOS (.app) — build on macOS

```bash
cd desktop
wails3 task package                    # .app for the current architecture
wails3 task darwin:package:universal   # universal .app (Apple Silicon + Intel)
```

The `.app` is ad-hoc signed, which is fine on your own machine. To distribute
to others, sign with a Developer ID certificate and notarize (configure once
with `wails3 setup`):

```bash
wails3 task darwin:sign:notarize
```

#### Windows (.exe + NSIS installer) — build on Windows or CI

```bash
cd desktop
wails3 task package
```

Produces `bin/<app>.exe` plus an NSIS installer that automatically installs
the WebView2 runtime if missing. Building the installer requires
[NSIS](https://nsis.sourceforge.io). For a release, sign the installer with
an Authenticode certificate:

```bash
wails3 task windows:sign:installer
```

#### Linux (deb / rpm / AppImage) — build on Linux

```bash
cd desktop
wails3 task package
```

Builds the binary plus deb and rpm packages (via nfpm) and an AppImage (the
AppImage step downloads the `linuxdeploy` tool on first run). The deb/rpm
declare GTK4 + WebKitGTK 6.0 as dependencies; for older distros build with
the legacy stack (`EXTRA_TAGS=gtk3`) and adjust `desktop/build/linux/nfpm/nfpm.yaml`.

### 5. Cross-compilation and CI

- Windows executables cross-compile from macOS/Linux without CGO:
  `wails3 task build GOOS=windows`.
- macOS and Linux builds use CGO, so release builds should run on the target
  OS — a GitHub Actions matrix (one job per OS) is the recommended setup — or
  use the Wails Docker cross image (`wails3 task setup:docker`, ~800 MB).
- Cross-built artifacts are unsigned; sign on the target OS before
  distributing.

### Data and migrations at runtime

Desktop builds apply your migrations automatically on every launch. Data
lives in the per-user config directory: `~/Library/Application Support/<name>`
on macOS, `%APPDATA%\<name>` on Windows, `~/.local/share/<name>` on Linux.
After adding migrations or plugins to the project, re-run
`airway desktop:init` to refresh the embedded copy.

The framework's [desktop guide](https://github.com/daqing/airway/blob/main/docs/desktop.md)
has the full background and troubleshooting notes.

## Acknowledgments

- [Airway](https://github.com/daqing/airway) — the Go web framework this
  project is built on.

## License

AirShop is released under the [MIT license](LICENSE).
