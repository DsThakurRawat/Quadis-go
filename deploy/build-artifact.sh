#!/usr/bin/env bash
#
# Build the deployable artifact — and ONLY the deployable artifact.
#
# Run from the repo root:  ./deploy/build-artifact.sh
# Produces:                dist-artifact/quadis-<git-sha>.tar.gz
#
# WHAT SHIPS, and nothing else:
#   www/            the built frontend (dist/), including public/ assets
#   api/            the compiled Go backend binary (api/server)
#   nginx/          the real server config and the 63 legacy 301s
#   systemd/        the API unit, which is what sets NODE_ENV=production
#
# WHAT DOES NOT SHIP, and why it is listed explicitly rather than left to a
# .gitignore-style guess:
#   docs/           client-comms/** is internal correspondence about the client
#   .agents/        AWS account ids, instance ids, incident history
#   client-assets/  423 files including her plaintext logins
#   backend/.env    a live Groq key
#   src/ scripts/   sources; the box runs the build output, not the repo
#   .git/           the whole history, on a public web root
#
# The rule this encodes: the artifact is ASSEMBLED, never synced. A
# `rsync repo-root -> /var/www` is the single mistake that turns all of the
# above into public URLs on the client's own domain. nginx has deny rules as a
# backstop (deploy/nginx/quadis.conf) but the backstop is not the plan.

set -euo pipefail

cd "$(dirname "$0")/.."
ROOT="$PWD"
SHA="$(git rev-parse --short HEAD 2>/dev/null || echo nogit)"
DIRTY=""
if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null; then
  DIRTY="-dirty"
fi
STAGE="$ROOT/dist-artifact/stage"
OUT="$ROOT/dist-artifact/quadis-go-${SHA}${DIRTY}.tar.gz"

echo "==> Building artifact for Quadis-go (${SHA}${DIRTY})"
rm -rf "$STAGE" && mkdir -p "$STAGE"/{www,api,nginx,systemd}

# --- frontend -------------------------------------------------------------
# `npm run build` runs prebuild -> generate-sitemap, so the sitemap in the
# artifact is always regenerated from src/data/hotels.ts rather than whatever
# stale copy happens to be in public/.
echo "==> Building frontend"
VITE_API_URL=/api npm run build >/dev/null
cp -r "$ROOT/dist/." "$STAGE/www/"

# Belt and braces: the internal design page was tracked under public/ once and
# would have been served at https://www.quadishotels.com/QuadisLocationandLinks.html.
# It has been moved to docs/, but assert rather than assume.
find "$STAGE/www" -maxdepth 1 -name "QuadisLocation*" -delete

# --- backend --------------------------------------------------------------
echo "==> Building backend (Go)"
mkdir -p "$STAGE/api"
( cd "$ROOT/backend-go" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$STAGE/api/server" ./cmd/server )
echo "==> Go backend compiled into $STAGE/api/server"

# --- config ---------------------------------------------------------------
cp "$ROOT/deploy/nginx/quadis.conf"            "$STAGE/nginx/"
cp "$ROOT/deploy/nginx/legacy-redirects.conf"  "$STAGE/nginx/"
cp "$ROOT/deploy/nginx/security-headers.conf"  "$STAGE/nginx/"
cp "$ROOT/deploy/quadis-api.service"           "$STAGE/systemd/"
cp "$ROOT/deploy/install.sh"                   "$STAGE/"
chmod +x "$STAGE/install.sh"

echo "$SHA$DIRTY" > "$STAGE/VERSION"

# --- refuse to ship anything sensitive ------------------------------------
# A tarball is opaque once built. Fail loudly here rather than discover it on
# the web server.
echo "==> Verifying artifact contains nothing it should not"
LEAKS="$(find "$STAGE" \( \
      -name ".env" -o -name ".env.*" -o -name "*.pem" -o -name "*.key" \
   -o -name "AGENTS.md" -o -name "*.sql.gz" \
   \) -not -path "*/node_modules/*" 2>/dev/null || true)"
# global-bundle.pem is Amazon's PUBLIC RDS CA bundle and the backend build
# copies it deliberately; it is not a secret.
LEAKS="$(echo "$LEAKS" | grep -v "global-bundle.pem" | grep -v '^$' || true)"
if [ -n "$LEAKS" ]; then
  echo "REFUSING TO PACKAGE — sensitive files staged:" >&2
  echo "$LEAKS" >&2
  exit 1
fi
for forbidden in docs .agents client-assets .git src scripts; do
  if [ -e "$STAGE/$forbidden" ]; then
    echo "REFUSING TO PACKAGE — '$forbidden' is staged" >&2
    exit 1
  fi
done

tar -czf "$OUT" -C "$STAGE" .
echo
echo "==> $OUT"
echo "    $(du -h "$OUT" | cut -f1)   contents:"
tar -tzf "$OUT" | awk -F/ '{print $2}' | sort -u | grep -v '^$' | sed 's/^/      /'
