#!/usr/bin/env bash
#
# Start the API and the site together, and stop both together.
#
# Running them in separate terminals is easy to get wrong: a stale API on 3001
# keeps serving old code while the new one dies with EADDRINUSE, and killing the
# frontend leaves the backend running.
#
#   ./scripts/dev.sh          both
#   ./scripts/dev.sh api      API only
#   ./scripts/dev.sh web      site only
#
set -euo pipefail
cd "$(dirname "$0")/.."
ROOT="$PWD"

API_PORT="${PORT:-3001}"
WEB_PORT="${WEB_PORT:-5173}"
WHAT="${1:-all}"

if [ "$WHAT" = "stop" ]; then
  for port in "$API_PORT" "$WEB_PORT" 5174; do
    command -v fuser >/dev/null 2>&1 && fuser -k "${port}/tcp" >/dev/null 2>&1 || true
  done
  echo "Stopped anything on ${API_PORT}, ${WEB_PORT}, 5174."
  exit 0
fi

# Local defaults so the app runs out of the box. Real secrets belong in
# backend/.env — which is gitignored — and override these.
export ADMIN_PIN="${ADMIN_PIN:-998877}"
export ADMIN_PASSWORD="${ADMIN_PASSWORD:-local-dev-token}"
export SESSION_SECRET="${SESSION_SECRET:-local-dev-session-secret}"

free_port() {
  local port="$1"
  if command -v fuser >/dev/null 2>&1 && fuser "${port}/tcp" >/dev/null 2>&1; then
    echo "  port ${port} was in use — freeing it"
    fuser -k "${port}/tcp" >/dev/null 2>&1 || true
    sleep 1
  fi
}

PIDS=()
cleanup() {
  echo ""
  echo "Stopping…"
  # Kill each child's whole process group, not just the shell that launched it:
  # `npx` spawns node as a grandchild, so killing the wrapper alone leaves the
  # server running and the port held.
  for pid in "${PIDS[@]:-}"; do
    [ -n "${pid:-}" ] || continue
    kill -TERM -- "-${pid}" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
  done
  sleep 1
  # Belt and braces: anything still holding a port goes now.
  for port in "$API_PORT" "$WEB_PORT"; do
    command -v fuser >/dev/null 2>&1 && fuser -k "${port}/tcp" >/dev/null 2>&1 || true
  done
  wait 2>/dev/null || true
}
# Stop everything on Ctrl-C, and whenever this script exits for any reason.
trap cleanup INT TERM EXIT

echo "Quadis-go dev"

if [ "$WHAT" = "all" ] || [ "$WHAT" = "api" ]; then
  free_port "$API_PORT"
  echo "  API (Go) → http://localhost:${API_PORT}/api"
  if [ -f "$ROOT/backend-go/bin/server" ]; then
    setsid env PORT="$API_PORT" "$ROOT/backend-go/bin/server" &
  else
    setsid env PORT="$API_PORT" bash -c 'cd "$0/backend-go" && exec go run ./cmd/server' "$ROOT" &
  fi
  PIDS+=($!)
fi

if [ "$WHAT" = "all" ] || [ "$WHAT" = "web" ]; then
  free_port "$WEB_PORT"
  echo "  Site → http://localhost:${WEB_PORT}/"
  setsid bash -c 'cd "$0" && exec npx vite --port "$1" --strictPort' "$ROOT" "$WEB_PORT" &
  PIDS+=($!)
fi

echo "  Ctrl-C stops both."
echo ""

# If either process dies, bring the other down too rather than leaving half the
# stack running and pointing at nothing.
wait -n
echo ""
echo "A process exited — shutting the rest down."
