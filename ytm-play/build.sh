#!/bin/sh
# Build ytm-play for the JBL Link Portable (ARMv7, glibc >= 2.4 on the device).
#
# The player links faad2 statically, built WITHOUT the DRM flag: the firmware's
# own libfaad_drm.so is compiled for Digital Radio Mondiale and rejects ordinary
# AAC (errors 29 and 30). Everything else is loaded at runtime via dlopen
# (libasound.so) or comes from libc.
#
# Requirements (versions used for the published binary):
#   - Zig 0.17.0            pip install ziglang==0.17.0
#   - git
#   - GNU binutils for ARM  arm-linux-gnueabihf-strip (Binutils 2.42)
#   - UPX 4.2.2             optional, only to shrink the binary
#
# With these versions the result is byte-identical to the released binary
# (SHA-256 a6f498c98ee2d0e29f3670095091311b1b9bbcd36f1378ba773ec992ce09a9f4).
set -eu

FAAD2_REPO=https://github.com/knik0/faad2.git
FAAD2_COMMIT=a335040022e14ff9b6c54bbd459590d4c7edffe8

here=$(cd "$(dirname "$0")" && pwd)
build=$here/build
mkdir -p "$build"

cc() {
    python3 -m ziglang cc --target=arm-linux-gnueabihf.2.27 -mcpu=cortex_a35 "$@"
}

if [ ! -d "$build/faad2" ]; then
    git clone -q "$FAAD2_REPO" "$build/faad2"
fi
git -C "$build/faad2" checkout -q "$FAAD2_COMMIT"

# faad2 normally gets these from its build system's config.h.
defs='-DHAVE_INTTYPES_H=1 -DHAVE_STDINT_H=1 -DHAVE_STRING_H=1 -DHAVE_STDLIB_H=1
      -DHAVE_UNISTD_H=1 -DHAVE_MEMCPY=1 -DPACKAGE_VERSION="2.11"'

rm -rf "$build/obj"
mkdir -p "$build/obj"
for src in "$build"/faad2/libfaad/*.c; do
    name=$(basename "$src" .c)
    # shellcheck disable=SC2086
    cc -Os -fno-strict-aliasing $defs -I"$build/faad2/libfaad" \
        -I"$build/faad2/include" -c "$src" -o "$build/obj/$name.o"
done
rm -f "$build/libfaad.a"
python3 -m ziglang ar rcs "$build/libfaad.a" "$build"/obj/*.o

cc -Os -Wall -o "$build/ytm-play" "$here/ytm-play.c" "$build/libfaad.a" \
    -I"$build/faad2/include" -ldl -lm -lpthread
arm-linux-gnueabihf-strip "$build/ytm-play"

if command -v upx >/dev/null 2>&1; then
    rm -f "$build/ytm-play.upx"
    upx -q --best --lzma -o "$build/ytm-play.upx" "$build/ytm-play" >/dev/null
    mv "$build/ytm-play.upx" "$build/ytm-play"
fi

sha256sum "$build/ytm-play"
