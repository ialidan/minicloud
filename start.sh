#!/usr/bin/env bash
set -euo pipefail

PORT="${MINICLOUD_PORT:-8080}"

echo "==> Building and starting MiniCloud..."
docker compose up -d --build

echo ""
echo "==> MiniCloud is running!"
echo ""
echo "    Local:   http://localhost:${PORT}"

# Detect LAN IP so devices on the same network (phone, tablet, etc.) can connect.
LAN_IP=""
if command -v ipconfig &>/dev/null; then
  # macOS
  for iface in en0 en1; do
    LAN_IP=$(ipconfig getifaddr "$iface" 2>/dev/null || true)
    [ -n "$LAN_IP" ] && break
  done
fi

if [ -z "$LAN_IP" ] && command -v ip &>/dev/null; then
  # Linux
  LAN_IP=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="src") print $(i+1)}' || true)
fi

if [ -z "$LAN_IP" ] && command -v hostname &>/dev/null; then
  LAN_IP=$(hostname -I 2>/dev/null | awk '{print $1}' || true)
fi

if [ -n "$LAN_IP" ]; then
  echo "    Network: http://${LAN_IP}:${PORT}  (use this on your phone)"
else
  echo "    Could not detect LAN IP. Check your network settings."
fi

echo ""
echo "    Logs:    docker compose logs -f minicloud"
echo "    Stop:    docker compose down"
