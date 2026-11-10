# Audio

## ALSA layout

Relevant parts of `/etc/asound.conf` on this firmware:

```
default ──► shadow_softvol (control "ShadowMaster") ──► ... ──► dmixer ──► hw:0,0
softvol ──► softvol (control "Master") ────────────────────────► dmixer
hmpp    ──► Harman DSP plugin (type hmpp) ──► ch2route ─────────► dmixer
```

- **Alice** speaks through `default`, so her volume is `ShadowMaster` (numid 120).
- The **volume buttons** move `Master` (numid 128) and `ShadowMaster` together,
  range 0 to 999.
- **Bluetooth** goes through `hmpp`, the Harman DSP. That is what gives the
  speaker its tuned bass; anything that bypasses it sounds noticeably flatter.
- `dmixer` is a `dmix` with shared memory keys 1024 and 1025.

## Our output device

ytm-play uses a device of our own, `ytmout`: a softvol on the `Master` control
feeding `plug:hmpp`. So the buttons control it, and it gets the same DSP as
Bluetooth.

The home directory `/home/root` is on the read-only root, so `~/.asoundrc` is not
an option. Instead the player gets `ALSA_CONFIG_PATH` pointing at
[device/ytm/my.conf](../device/ytm/my.conf), which must include **both** the main
ALSA config and the system one before defining `ytmout`:

```
</usr/share/alsa/alsa.conf>
</etc/asound.conf>
pcm.ytmout { type softvol  slave.pcm "plug:hmpp"  control { name "Master" card 0 } ... }
```

Without the first include, ALSA does not know the `plug` type at all.

Keeping Alice on `ShadowMaster` and the music on `Master` also lets ytm-alice mute
only Alice for a moment while the music keeps playing; see
[10](10-voice-control.md).

## Stale dmix attachments

If a player using `dmixer` is killed hard, its attachment to the shared memory
segments can stay behind. A few of those and Bluetooth goes silent. BusyBox has no
`ipcrm`, hence [tools/ipcclean.c](../tools/ipcclean.c). Recovery:

```
/data/ytm/bin/ipcclean dmix
systemctl restart linkplay.service      # plays the JBL chime
```

ytm-play releases its slot from an `atexit` handler and closes its socket from the
signal handler, so this should not happen with it any more.

## What the firmware cannot decode

YouTube Music delivers AAC in MP4 (progressive or fragmented). On this firmware:

- GStreamer has no AAC decoder and no MP4 demuxer; MP3 works through libmad.
- `libquasar_gstreamer.so` contains such elements, but exports none of them.
- `/system/workdir/lib/libfaad_drm.so` exports the faad2 API, but it is built
  for Digital Radio Mondiale and rejects ordinary AAC with errors 29 and 30.

Hence our own player with its own faad2: [08](08-ytm-play.md).

## The boot-time gain bug

`HW Gain1 Volume` (numid 7) is the codec's hardware gain, normally `524288`.

On a speaker that has just booted, reading the simple mixer element with

```
amixer -c0 sget Master
```

sets it to `64`: the speaker is effectively silent, and Alice too, until
`linkplay.service` is restarted. After such a restart the same command is
harmless, which is why it hid for so long: every manual test had been done right
after a restart that had been needed to recover the sound.

`HW Gain1 Volume` cannot be put back with `amixer cset` either; it reads back as
`64` immediately. Only the Linkplay start-up sets it.

Reading or writing the control **element** directly does not trigger it:

```
amixer -c0 cget numid=128        # or name=Master
amixer -c0 cset numid=128 500,500
```

ytm-alice used `sget`/`sset Master` to report and set the volume for the cast
session. The patch replaces them with element access.

## Finding such things: compare the mixer

```
amixer -c 0 contents > mix-bad.txt
# recover, then
amixer -c 0 contents > mix-good.txt
diff mix-bad.txt mix-good.txt
```

Out of 645 lines, five differed, and one of them was the gain above.
