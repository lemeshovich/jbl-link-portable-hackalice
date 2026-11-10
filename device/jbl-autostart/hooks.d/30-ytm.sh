#!/bin/sh
set -eu
[ "$(id -u)" = 0 ] || exit 1
[ ! -e /data/jbl-autostart/ytm-off ] || { echo "ytm disabled"; exit 0; }
[ -x /data/ytm/bin/ytm-alice ] || exit 0

# /etc is read-only, so install the unit into /run and let systemd order it
# after linkplay.service. The hook only drops the file and returns at once.
mkdir -p /run/systemd/system
cat > /run/systemd/system/ytm-alice.service <<'UNIT'
[Unit]
Description=ytm-alice YouTube Music for Alice
After=linkplay.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStartPre=/bin/sh -c "chmod 600 /data/yam/var/local.ytm-alice/*.json 2>/dev/null; true"
ExecStart=/data/ytm/bin/ytm-alice daemon start
ExecStop=/data/ytm/bin/ytm-alice daemon stop
WorkingDirectory=/data/ytm/bin
UNIT
systemctl daemon-reload >/dev/null 2>&1 || true
systemctl start --no-block ytm-alice.service >/dev/null 2>&1 || true
echo "ytm-alice unit installed"
