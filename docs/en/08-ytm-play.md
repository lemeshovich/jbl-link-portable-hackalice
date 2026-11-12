# ytm-play

**English** · [Українська](../uk/08-ytm-play.md) · [Русский](../ru/08-ytm-play.md)

A small AAC player for this speaker, written because the firmware cannot decode
what YouTube Music sends ([07](07-audio.md#what-the-firmware-cannot-decode)).
Source: [ytm-play/ytm-play.c](../../ytm-play/ytm-play.c). License: GPL-2.0, because
faad2 is linked in.

## What it does

```
http:// stream or local file ─► MP4 parser ─► faad2 (AAC) ─► ALSA (via dlopen)
```

- Input: a plain `http://` URL or a local file. ytm-alice serves the audio to it
  from a local relay at `127.0.0.1:18883`, so no TLS is needed in the player.
- MP4: progressive files (`moov` before `mdat`) and fragmented ones (`moof` with
  `mdat`; other boxes such as `sidx` are skipped).
- Decoding: faad2 built **without** the DRM flag and linked statically.
- Output: `libasound.so` loaded with `dlopen`, so the binary does not depend on the
  firmware's ALSA headers or exact library version. Device chosen with `-D`
  (on the speaker: `ytmout`).

## Interface used by ytm-alice

```
ytm-play [-v] [-g 0..100] [-D alsa-device] <url|file>
```

Standard input, one command per line:

| Command | Effect |
|---|---|
| `pause` / `resume` | pause or continue |
| `volume <0..1>` | software gain, used to duck the music while Alice talks |
| `stop` | stop and exit |

Standard output, one JSON object per line: `{"event":"playing"}`, `paused`,
`position`, `eos`, `error`.

`-g 0` plays silently, which is handy for testing a device without hearing it.

## Shutting down cleanly

- The signal handler closes the network socket at once, so a read blocked on a
  stalled stream does not keep the process alive.
- An `atexit` handler drains and closes the PCM, so the `dmix` slot is released
  (see [stale attachments](07-audio.md#stale-dmix-attachments)).

## Build

[ytm-play/build.sh](../../ytm-play/build.sh): Zig 0.17.0 targeting
`arm-linux-gnueabihf.2.27` and Cortex-A35, faad2 at commit
`a335040022e14ff9b6c54bbd459590d4c7edffe8`, stripped and packed with UPX 4.2.2.
The result is byte-identical to the released binary.
