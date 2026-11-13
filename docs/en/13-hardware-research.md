# Hardware research notes

**English** · [Українська](../uk/13-hardware-research.md) · [Русский](../ru/13-hardware-research.md)

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

`/sys/class/leds/` has `power-white`, `red`, `white1`…`white5` and `wifi`.
`white1`…`white5` are the **battery indicator on the back**, not the ring at the
front.

The front ring (purple while Alice listens, spins while she thinks, blue for
Bluetooth, red for muted microphones) is driven by quasar itself through
`/dev/i2c-led`, which `maind` keeps open. Its protocol is unknown and there is
no IPC message to drive it; the only `QuasarMessage` about LEDs,
`led_screen_command` (180), is for screen capture. Writing to the device in
parallel with quasar would risk confusing its state.

## Linkplay

`linkplay.sh` sets the codec routing with `amixer` (switches such as
`O03 I05`, `I2S O03_O04`, `AmpSwitch`) and then starts `rootApp`. Re-running only
those `amixer` lines does **not** restore lost sound; the gain is set later by the
Linkplay start-up itself.
