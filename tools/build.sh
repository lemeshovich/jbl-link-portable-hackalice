#!/bin/sh
# Build the diagnostic tools for the JBL Link Portable (ARMv7, glibc >= 2.4).
#
# Requirements: Zig 0.17.0 (pip install ziglang==0.17.0) and GNU binutils for
# ARM (arm-linux-gnueabihf-strip, Binutils 2.42). With these versions the
# results are byte-identical to the released binaries; see SHA256SUMS.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$here/build"
for tool in alicesay alicedump qcancel ipcclean qplay; do
    python3 -m ziglang cc --target=arm-linux-gnueabihf.2.27 -mcpu=cortex_a35 \
        -Os -Wall -o "$here/build/$tool" "$here/$tool.c"
    arm-linux-gnueabihf-strip "$here/build/$tool"
done
(cd "$here/build" && sha256sum alicesay alicedump qcancel ipcclean qplay)
