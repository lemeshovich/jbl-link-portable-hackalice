#!/bin/sh
# Build the Fluent Bit 1.4.6 input plugin from fluent_bootstrap.c (PR #141).
#
# PR #141 builds it with clang and ELF lld; this script uses Zig instead, which
# needs no separate ARM sysroot. Requirements: Zig 0.17.0
# (pip install ziglang==0.17.0) and arm-linux-gnueabihf-strip (Binutils 2.42).
# With these versions the result is byte-identical to the plugin used on the
# device (SHA-256 ea33c78f3f709b2d1de177cc9b8d43c0a86527d2fe8c2f3d3e26909ce0f577d5).
set -eu
here=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$here/build"
python3 -m ziglang cc --target=arm-linux-gnueabihf.2.27 -mcpu=cortex_a35 \
    -fPIC -O2 -shared -o "$here/build/flb-in_jbl_bootstrap.so" "$here/fluent_bootstrap.c"
arm-linux-gnueabihf-strip --strip-all "$here/build/flb-in_jbl_bootstrap.so"
sha256sum "$here/build/flb-in_jbl_bootstrap.so"
