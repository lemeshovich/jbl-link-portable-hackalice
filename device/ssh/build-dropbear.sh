#!/bin/sh
# Build a static, key-only Dropbear for the JBL Link Portable (ARMv7).
#
# Static musl via Zig: the firmware's glibc has no usable NSS setup for an SSH
# server, and a static binary does not depend on the device's libraries at all.
# Password and PAM authentication are compiled out.
#
# Requirements: Zig 0.17.0 (pip install ziglang==0.17.0), git, make,
# arm-linux-gnueabihf-strip (Binutils 2.42).
# Unlike the other binaries in this repository, Dropbear is NOT reproduced
# byte-for-byte: the copy running on the device (SHA-256
# 7e7d7aad828dcf31fe3bab8b436a90d910c9968c8e470380dbc916f5740d93e8, published
# in the release) was built iteratively from the same commit and options.
set -eu
DROPBEAR_REPO=https://github.com/mkj/dropbear.git
DROPBEAR_COMMIT=17549ea6b6be0d59e99c5c5350185c0f9cc57211

here=$(cd "$(dirname "$0")" && pwd)
build=$here/build
mkdir -p "$build/bin"
cat > "$build/bin/zigcc" <<'W'
#!/bin/sh
exec python3 -m ziglang cc --target=arm-linux-musleabihf -mcpu=cortex_a53 "$@"
W
printf '#!/bin/sh\nexec python3 -m ziglang ar "$@"\n' > "$build/bin/zigar"
printf '#!/bin/sh\nexec python3 -m ziglang ranlib "$@"\n' > "$build/bin/zigranlib"
chmod 755 "$build"/bin/*
PATH=$build/bin:$PATH

if [ ! -d "$build/dropbear" ]; then
    git clone -q "$DROPBEAR_REPO" "$build/dropbear"
fi
cd "$build/dropbear"
git checkout -q -f "$DROPBEAR_COMMIT"
git clean -qfdx
cat > localoptions.h <<'O'
#define DROPBEAR_SVR_PASSWORD_AUTH 0
#define DROPBEAR_SVR_PAM_AUTH 0
#define DROPBEAR_SVR_PUBKEY_AUTH 1
#define DROPBEAR_USE_PASSWORD_ENV 0
O
./configure --host=arm-linux-musleabihf --disable-zlib --disable-wtmp \
    --disable-lastlog CC=zigcc AR=zigar RANLIB=zigranlib \
    CFLAGS="-Os -static" LDFLAGS="-static" >/dev/null
make -j4 PROGRAMS="dropbear dropbearkey" STATIC=1 >/dev/null
mkdir -p "$build/out"
cp dropbear dropbearkey "$build/out/"
arm-linux-gnueabihf-strip --strip-all "$build/out/dropbear" "$build/out/dropbearkey"
(cd "$build/out" && sha256sum dropbear dropbearkey)
