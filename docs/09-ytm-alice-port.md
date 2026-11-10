# ytm-alice port

[ytm-alice](https://github.com/SPAWNRYS-ban/ytm-alice) is a YouTube Music client
for Yandex speakers that listens to Alice and takes over music commands. It is
written for newer firmware. This page describes how it was made to run on quasar
3.21; [10](10-voice-control.md) describes the voice-control part in detail.

## Binaries

| File | What |
|---|---|
| `ytm-alice` | the daemon and CLI, Go, built with the patch from `patches/` |
| `ytm-alice-resolver` | QuickJS-ng based helper that deciphers stream URLs, static |
| `ytm-alice-player` | shell wrapper, see [device/ytm/ytm-alice-player](../device/ytm/ytm-alice-player) |
| `ytm-play` | the actual player, see [08](08-ytm-play.md) |

All four go into `/data/ytm/bin/`. The ALSA config goes to `/data/ytm/my.conf`.

ytm-alice starts `ytm-alice-player` from its own directory with an
`http://127.0.0.1:18883/audio/...` URL; the wrapper sets `ALSA_CONFIG_PATH` and
runs `ytm-play -D ytmout`. The firmware's GStreamer is not used at all.

Build: [patches/build-ytm-alice.sh](../patches/build-ytm-alice.sh) clones
ytm-alice at commit `923e9acc2517184c447853e00e96eb600f5cef94`, applies
[the patch](../patches/ytm-alice-quasar-3.21.patch) and builds both programs
(Go 1.27.3, `arm-linux-gnueabihf-gcc` 13.3.0 for the resolver, UPX 4.2.2). The
output is byte-identical to the released binaries.

## First start

```
cd /data/ytm/bin
./ytm-alice login tv          # prints a code and the address to enter it at
./ytm-alice daemon start
./ytm-alice status
```

The TV login does not expire like a browser session. The web panel is on port
24721 and the speaker appears as a cast target named "Яндекс Станция".

## Autostart

[device/jbl-autostart/hooks.d/30-ytm.sh](../device/jbl-autostart/hooks.d/30-ytm.sh)
writes `ytm-alice.service` into `/run/systemd/system` and starts it without
waiting. The unit:

- is ordered after `linkplay.service` and the network;
- runs `chmod 600` on the files in `/data/yam/var/local.ytm-alice/` first,
  because ytm-alice refuses a session file that is not mode 600, and
  `droproot_setup` resets it at every boot ("файл сессии должен иметь права 600");
- starts the daemon with no delay. Delays were never the fix
  ([03](03-persistence-and-autostart.md#the-boot-time-silence-that-was-not-about-autostart)).

## What the patch changes

Against upstream commit `923e9acc`, in `main.go`, `ipc.go`, `cast.go`, plus a test
file. Everything firmware-specific is enabled only when `maind` does not contain
`AliceCapability` (or when `YTM_ALICE_SPEECH=server_action` is set), so newer
firmware keeps the original behaviour.

| Change | Why |
|---|---|
| Carry the recognised text from the state 1 frame to the textless state 3 frame | otherwise no voice command is ever recognised on 3.21 |
| Speak through `vins_server_action` + `repeat_after_me` | `AliceCapability` speech does not exist on 3.21; gives back "Включаю" and "что сейчас играет" |
| Drop Alice's refusal by its text in state 3 with `cleanup_directive_controller`, then release the dialog | the cloud answers "video not supported" or "subscription required" to every intercepted command |
| Mute only Alice (`ShadowMaster`) while a command is being taken over | a dropped reply sometimes still reaches the speaker for a few hundred ms |
| Send the announcement only after the refusal was dropped, and resend once if Alice did not pick it up | the cancel also cancels a request in flight; right after a cancel `aliced` sometimes ignores one request |
| Close the microphone that opens by itself after our requests | otherwise Alice listens for about 5 s and the music stays ducked |
| Keep IPC connections open 0.5 s after writing | `aliced` drops messages from connections that close too early |
| A repeated phrase is a new command if a wake word came in between | "дальше" twice within 10 s was ignored as a duplicate frame |
| Read and set the volume through the `Master` control element instead of `amixer sget/sset Master` | the simple-mixer access silences the speaker after boot ([07](07-audio.md#the-boot-time-gain-bug)) |
| No announcement on "дальше" / "назад" | upstream announces on track change too |
| `/tmp/ytm-voice.log` | a journal of Alice states and of every action taken, small and rotated |

Tests for the frame decoding, refusal detection and the announcement gate are in
`legacy_voice_test.go`; the upstream test suite passes.
