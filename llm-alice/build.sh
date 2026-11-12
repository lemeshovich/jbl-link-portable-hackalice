#!/bin/sh
# Build llm-alice for the JBL Link Portable (ARMv7). Standard library only.
#
# Requirements: Go 1.27.3, UPX 4.2.2 (optional). With these versions the result
# is byte-identical to the released binary; see SHA256SUMS.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$here/build"
cd "$here"
go test ./...
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 \
    go build -trimpath -ldflags '-s -w' -o "$here/build/llm-alice.bin" .
if command -v upx >/dev/null 2>&1; then
    rm -f "$here/build/llm-alice"
    upx -q --best --lzma -o "$here/build/llm-alice" "$here/build/llm-alice.bin" >/dev/null
    rm -f "$here/build/llm-alice.bin"
else
    mv "$here/build/llm-alice.bin" "$here/build/llm-alice"
fi
sha256sum "$here/build/llm-alice"
