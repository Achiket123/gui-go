#!/usr/bin/env bash
# scripts/dev.sh — hot-reload wrapper for goui-counter
#
# How it works:
#   1. Builds the binary with the `debug` tag (enables devtools + HotReloader)
#   2. Runs the binary
#   3. If the binary exits with code 0 (triggered by HotReloader.OnChange),
#      jumps back to step 1 — giving you a full hot-reload loop.
#   4. Any other exit code (crash, Ctrl+C) stops the loop.
#
# Usage:
#   chmod +x scripts/dev.sh
#   ./scripts/dev.sh
#
# Optional env vars:
#   GOUI_WATCH_DIR   — directory to watch (default: project root)
#   GOUI_BUILD_FLAGS — extra go build flags

set -euo pipefail

BINARY="./bin/counter-debug"
BUILD_TAGS="debug"
MAIN_PKG="./cmd/counter"

build() {
  echo "🔨 Building…"
  go build \
    -tags "${BUILD_TAGS}" \
    ${GOUI_BUILD_FLAGS:-} \
    -o "${BINARY}" \
    "${MAIN_PKG}"
  echo "✅ Build OK"
}

mkdir -p ./bin

# Initial build — fail fast if there are errors before we start watching
build

while true; do
  echo "🚀 Starting counter app…"
  "${BINARY}" "$@"
  EXIT_CODE=$?

  if [ "${EXIT_CODE}" -eq 0 ]; then
    # Exit 0 = HotReloader triggered a rebuild + restart
    echo "🔄 Source changed — rebuilding…"
    if build; then
      echo "🔁 Restarting…"
      # loop continues → binary re-runs
    else
      echo "❌ Build failed — waiting for next change…"
      # Don't exit; let the developer fix the error and save again.
      # The running binary already watches for the next change.
      sleep 1
    fi
  else
    # Non-zero = crash or Ctrl+C → stop the loop
    echo "⛔ App exited with code ${EXIT_CODE} — stopping."
    exit "${EXIT_CODE}"
  fi
done
