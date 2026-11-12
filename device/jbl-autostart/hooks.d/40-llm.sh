#!/bin/sh
set -eu
[ "$(id -u)" = 0 ] || exit 1
[ ! -e /data/jbl-autostart/llm-off ] || { echo "llm disabled"; exit 0; }
[ -x /data/llm/llm-alice ] && [ -s /data/llm/key ] || exit 0

# /etc is read-only: install the unit into /run and return at once.
mkdir -p /run/systemd/system
cat > /run/systemd/system/llm-alice.service <<'UNIT'
[Unit]
Description=llm-alice LLM answers in Alice's voice
After=linkplay.service network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStartPre=/bin/sh -c "chmod 600 /data/llm/key 2>/dev/null; true"
ExecStart=/data/llm/llm-alice serve
Restart=on-failure
RestartSec=5
UNIT
systemctl daemon-reload >/dev/null 2>&1 || true
systemctl start --no-block llm-alice.service >/dev/null 2>&1 || true
echo "llm-alice unit installed"
