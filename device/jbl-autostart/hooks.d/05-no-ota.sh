#!/bin/sh
set -eu
[ "$(id -u)" = 0 ] || exit 1
for t in /system/vendor/quasar/device_ota_update.sh /system/vendor/quasar/a01localupdate; do
    [ -e "$t" ] || continue
    mount | grep -q " $t " || mount -o bind /bin/true "$t"
done
echo "OTA neutralised"
