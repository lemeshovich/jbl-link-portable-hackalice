#!/bin/sh
# Build ytm-alice and ytm-alice-resolver with the quasar 3.21 patch, for the
# JBL Link Portable (ARMv7).
#
# Requirements (versions used for the published binaries):
#   - Go 1.27.3
#   - git, python3
#   - arm-linux-gnueabihf-gcc 13.3.0 (Ubuntu 24.04 gcc-arm-linux-gnueabihf)
#   - UPX 4.2.2
#
# golang.org/x/sys and golang.org/x/text were fetched from their official
# GitHub mirrors; Go records that in the binary, so the same replace lines are
# added here to get a byte-identical result. The module contents are the same.
set -eu

UPSTREAM=https://github.com/SPAWNRYS-ban/ytm-alice.git
UPSTREAM_COMMIT=923e9acc2517184c447853e00e96eb600f5cef94

here=$(cd "$(dirname "$0")" && pwd)
build=$here/build
mkdir -p "$build"

out=$build/out
mkdir -p "$out"
if [ ! -d "$build/src" ]; then
    git clone -q "$UPSTREAM" "$build/src"
fi
cd "$build/src"
git checkout -q -f "$UPSTREAM_COMMIT"
git clean -qfdx
git apply "$here/ytm-alice-quasar-3.21.patch"

cd source
cat >> go.mod <<'MOD'

replace golang.org/x/sys => github.com/golang/sys v0.42.0
replace golang.org/x/text => github.com/golang/text v0.35.0
MOD
export GOFLAGS=-mod=mod
go mod download
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 \
    go build -trimpath -ldflags '-s -w' -o "$out/ytm-alice.bin" .

# Resolver: QuickJS-ng amalgamation, linked statically.
python3 native/fetch-quickjs.py
cp native/.build/quickjs-amalgam.c native/.build/quickjs.h native/
(cd native && arm-linux-gnueabihf-gcc -O2 -static -D_GNU_SOURCE resolver.c \
    quickjs-amalgam.c -lm -ldl -pthread -latomic -s -o "$out/ytm-alice-resolver.bin")

for f in ytm-alice ytm-alice-resolver; do
    rm -f "$out/$f"
    upx -q --best --lzma -o "$out/$f" "$out/$f.bin" >/dev/null
    rm -f "$out/$f.bin"
done
(cd "$out" && sha256sum ytm-alice ytm-alice-resolver)
