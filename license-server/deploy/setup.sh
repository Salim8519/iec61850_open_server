#!/usr/bin/env bash
# Install or update the ScadaBench license server on Linux (Debian/Ubuntu/RHEL). Safe to re-run.
#   sudo bash deploy/setup.sh                 interactive first-time questions
#   sudo SET_ARGS="--listen 127.0.0.1:18090 --public-url https://..." bash deploy/setup.sh
#                                             no questions (used by deploy-server.cmd on Windows)
#   sudo bash deploy/setup.sh --configure     re-run the questions (PayPal, Mailgun secrets)
# Uses bin/license-server-linux-<arch> when present (built on Windows) so the server never compiles.
# A tunnel token in /etc/scadabench/tunnel.env (TUNNEL_TOKEN=...) enables the scadabench-tunnel service.
# Only ScadaBench files, users and services are touched; other software on the host is left alone.
set -euo pipefail

APP_USER=substation-license
APP_DIR=/opt/substation-license
DATA_DIR=/var/lib/substation-license
UNIT=/etc/systemd/system/license-server.service
TUNNEL_UNIT=/etc/systemd/system/scadabench-tunnel.service
TUNNEL_ENV=/etc/scadabench/tunnel.env
GO_VERSION=1.25.6
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE="$(cd "$HERE/.." && pwd)"
RECONFIGURE=0
[[ "${1:-}" == "--configure" ]] && RECONFIGURE=1

say() { printf '\n\033[1m==> %s\033[0m\n' "$*"; }
die() { printf '\nERROR: %s\n' "$*" >&2; exit 1; }
[[ $EUID -eq 0 ]] || die "run with sudo"
command -v systemctl >/dev/null || die "systemd is required"

case "$(uname -m)" in
  x86_64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;; *) die "unsupported CPU $(uname -m)" ;;
esac

binary() {
  local prebuilt="$SOURCE/bin/license-server-linux-$ARCH"
  if [[ -f "$prebuilt" ]]; then
    say "Using the program built on Windows ($ARCH)"; cp "$prebuilt" /tmp/license-server.new; return
  fi
  say "Building on this server (downloads Go once if needed)"
  local go=go
  if ! command -v go >/dev/null || ! go version | grep -Eq 'go1\.(2[4-9]|[3-9][0-9])'; then
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" -o /tmp/go.tgz
    rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tgz && rm -f /tmp/go.tgz
    go=/usr/local/go/bin/go
  fi
  (cd "$SOURCE" && CGO_ENABLED=0 GOFLAGS=-trimpath "$go" build -ldflags="-s -w" -o /tmp/license-server.new ./cmd/license-server)
}

binary
say "Installing"
id "$APP_USER" >/dev/null 2>&1 || useradd --system --home "$DATA_DIR" --shell /usr/sbin/nologin "$APP_USER"
install -d -m 0755 "$APP_DIR"
install -d -m 0700 -o "$APP_USER" -g "$APP_USER" "$DATA_DIR"
systemctl stop license-server 2>/dev/null || true
install -m 0755 /tmp/license-server.new "$APP_DIR/license-server" && rm -f /tmp/license-server.new
install -m 0644 "$HERE/license-server.service" "$UNIT"
cat > /usr/local/bin/license-server-admin <<EOF
#!/bin/sh
# Run license-server commands as the service user: sudo license-server-admin list
exec runuser -u $APP_USER -- env LICENSE_DATA_DIR=$DATA_DIR $APP_DIR/license-server "\$@"
EOF
chmod 0755 /usr/local/bin/license-server-admin

if [[ -n "${SET_ARGS:-}" ]]; then
  say "Applying settings"
  # shellcheck disable=SC2086
  license-server-admin set $SET_ARGS
fi
if [[ $RECONFIGURE -eq 1 || ( ! -f "$DATA_DIR/config.json" && -z "${SET_ARGS:-}" ) ]]; then
  say "Setup questions (press Enter to keep the value in brackets)"
  license-server-admin configure
fi

say "Starting the license server"
systemctl daemon-reload
systemctl enable license-server >/dev/null
systemctl restart license-server
sleep 1
systemctl is-active --quiet license-server || { journalctl -u license-server -n 30 --no-pager; die "service did not start"; }
LISTEN="$(grep -o '"listen_addr": *"[^"]*"' "$DATA_DIR/config.json" | sed 's/.*"\([^"]*\)"$/\1/')"
curl -fsS "http://${LISTEN}/api/v1/health" >/dev/null && echo "Health check: OK (http://${LISTEN})"

if [[ -f "$TUNNEL_ENV" ]]; then
  say "Cloudflare Tunnel (scadabench-tunnel service)"
  CLOUDFLARED=/usr/local/bin/cloudflared-scadabench
  if [[ ! -x "$CLOUDFLARED" ]]; then
    curl -fsSL "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-${ARCH}" -o "$CLOUDFLARED"
    chmod 0755 "$CLOUDFLARED"
  fi
  sed -i '1s/^\xEF\xBB\xBF//' "$TUNNEL_ENV"
  chmod 0600 "$TUNNEL_ENV"
  cat > "$TUNNEL_UNIT" <<EOF
[Unit]
Description=ScadaBench license server tunnel (Cloudflare)
After=network-online.target license-server.service
Wants=network-online.target

[Service]
EnvironmentFile=$TUNNEL_ENV
ExecStart=$CLOUDFLARED --no-autoupdate tunnel run
Restart=always
RestartSec=5
DynamicUser=yes
NoNewPrivileges=true
MemoryHigh=48M
MemoryMax=64M
CPUQuota=25%
TasksMax=64
Nice=5

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable scadabench-tunnel >/dev/null
  systemctl restart scadabench-tunnel
  sleep 3
  if systemctl is-active --quiet scadabench-tunnel; then echo "Tunnel: running"; else journalctl -u scadabench-tunnel -n 20 --no-pager; fi
fi

say "Done"
license-server-admin status
