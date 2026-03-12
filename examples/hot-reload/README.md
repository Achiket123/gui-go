# goui-counter

A minimal counter application built on `gui-go`, demonstrating:

- `state.Signal` for reactive state
- `state.History` for undo/redo
- `devtools.HotReloader` for live reload during development
- `devtools.LayoutDebugger` and `devtools.FPSOverlay` gated behind `//go:build debug`

---

## Project Structure

```
goui-counter/
├── cmd/counter/
│   ├── main.go            # Production entry point (no devtools)
│   └── main_debug.go      # Debug entry point (hot reload + devtools)
├── internal/ui/
│   ├── counter_app.go     # App, CounterWidget, ButtonBar
│   └── counter_test.go    # State + history tests
├── scripts/
│   └── dev.sh             # Hot-reload wrapper script
├── Makefile
└── go.mod
```

---

## Running the app

### Production

```bash
make run
# or
go build -o ./bin/counter ./cmd/counter && ./bin/counter
```

No devtools. No file watcher. Minimal binary.

---

## Hot Reload — Step by Step

Hot reload works by combining two things:

1. **`HotReloader`** (inside the binary) — polls `*.go` files every 500ms,
   runs `go build` when it detects a change, then calls `os.Exit(0)`.
2. **`scripts/dev.sh`** (wrapper script) — catches exit code 0 and
   re-launches the freshly-built binary.

### Start the dev loop

```bash
make dev
# which runs: ./scripts/dev.sh
```

You'll see:

```
🔨 Building…
✅ Build OK
🚀 Starting counter app…
[app] counter app running — press +/- or arrow keys, R to reset, Z to undo
[devtools] F1 → layout overlay | watch: /path/to/goui-counter
```

### Edit a file and save

```bash
# In another terminal, change anything — e.g. tweak a colour in counter_app.go
```

Within ~500ms you'll see in the first terminal:

```
[hot-reload] ✅ build succeeded — restarting…
🔄 Source changed — rebuilding…
🔨 Building…
✅ Build OK
🔁 Restarting…
🚀 Starting counter app…
```

The binary was rebuilt and restarted automatically. No manual `Ctrl+C`.

### What happens on a build error

If you introduce a syntax error:

```
🔄 Source changed — rebuilding…
🔨 Building…
# github.com/achiket123/goui-counter/internal/ui
internal/ui/counter_app.go:42:3: undefined: canvas.Bogus
❌ Build failed — waiting for next change…
```

The loop **does not exit** — it waits for you to fix the error and save again.

---

## Devtools (debug mode only)

These are only compiled when you use `-tags debug`.

| Feature | How to use |
|---|---|
| **LayoutDebugger** | Press `F1` in-app to toggle coloured bounds overlay |
| **FPSOverlay** | Always visible bottom-right; green < 16ms, yellow < 33ms, red otherwise |
| **EventLogger** | Every click/keypress printed to the terminal with prefix `[counter]` |
| **HotReloader** | Automatic — just save a `.go` file |

---

## Controls

| Key / Action | Effect |
|---|---|
| `+` or `ArrowUp` | Increment |
| `-` or `ArrowDown` | Decrement |
| `R` | Reset to 0 |
| `Z` | Undo last change |
| `Scroll up/down` | Increment / decrement |
| `F1` *(debug only)* | Toggle layout bounds overlay |

---

## How HotReloader is wired (annotated)

```go
// main_debug.go

reloader := devtools.NewHotReloader(projectRoot, func() {
    // OnChange is called only AFTER a successful build.
    log.Println("[hot-reload] ✅ build succeeded — restarting…")
    os.Exit(0)  // ← exit(0) signals the wrapper script to restart us
})

// BuildCmd is run before OnChange.
// Must be an allowlisted binary (go, make, task, fvm, shellcheck).
reloader.BuildCmd = []string{
    "go", "build",
    "-tags", "debug",
    "-o", os.Args[0],      // overwrite the current binary in place
    "./cmd/counter",
}

go reloader.Watch()  // starts polling in a background goroutine
```

```bash
# scripts/dev.sh (simplified)

build && \
while ./bin/counter-debug; do   # exit 0 → loop, non-0 → stop
    build
done
```

---

## Running tests

```bash
make test
# or
go test ./...
```
