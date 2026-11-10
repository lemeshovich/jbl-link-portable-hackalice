# Hardware research notes

Notes for anyone who wants to go further, for example to replace Alice with a
local voice stack. None of this is used by the current setup.

## Microphones

- Capture device `hw:0,1` (Wake on Voice, `pcmC0D1c`), `S32_LE`, 16000 Hz, two
  channels, period 1000, buffer 8000.
- It is held open by `maind`, so another program cannot capture while quasar
  runs.

## Playback loopback

- `hw:0,5` (AWB_Record) captures what the speaker plays, not the microphones.
  Useful to record output, or as an echo reference.

## Volume controls

| Control | numid | Range | Notes |
|---|---|---|---|
| `Master` | 128 | 0 to 999 | softvol, moved by the buttons |
| `ShadowMaster` | 120 | 0 to 999 | softvol of `default`, Alice's voice |
| `Digital Playback Volume` | 18 | 0 to 255 | default `207,207` |
| `HW Gain1 Volume` | 7 | normally 524288 | set by Linkplay at start; see [07](07-audio.md) |

## LEDs

```
/sys/class/leds/{power-white,red,white1..white5,wifi}/brightness
```

Writable by root. Alice drives them too, so expect her to overwrite them.

## Linkplay

`linkplay.sh` sets the codec routing with `amixer` (switches such as
`O03 I05`, `I2S O03_O04`, `AmpSwitch`) and then starts `rootApp`. Re-running only
those `amixer` lines does **not** restore lost sound; the gain is set later by the
Linkplay start-up itself.
