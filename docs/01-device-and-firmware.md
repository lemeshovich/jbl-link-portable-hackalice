# Device and firmware

## Hardware and system

| | |
|---|---|
| SoC | MediaTek MT8167, four Cortex-A35 cores, 32-bit ARMv7 userspace |
| RAM | about 505 MB |
| Kernel | 4.4.22 |
| Userland | Yocto, systemd, BusyBox 1.24.1 (no `pgrep`, `strings -o` prints **octal** offsets) |
| Firmware | `1.80.0.16.1086317237.20210928`, quasar 3.21 (`/system/vendor/quasar/version_full`) |

## Storage

- The root filesystem is UBIFS and **read-only**; `mount -o remount,rw` fails.
  `/etc`, `/home/root` and `/system` cannot be written. Bind mounts over existing
  files do work (see [05](05-overriding-the-firmware.md)).
- `/data` is a small writable UBIFS volume: about 17 MB in total, a few MB free.
  Everything persistent lives there. Budget space: the ytm-alice binaries alone
  take about 5 MB.
- `/tmp` and `/run` are tmpfs in RAM (about 250 MB), cleared on reboot.
- `droproot_setup.service` runs `droproot_setup.sh` with `Restart=always` and
  resets ownership and permissions under `/data` at every boot. Files that must
  keep their mode either get `chattr +i` or have it restored right before use.

## The two software stacks

The speaker runs two largely independent systems.

**Yandex quasar** (`/system/vendor/quasar/`): one `maind` binary started in many
roles by `quasar_launcher2`, for example `maind --yiod` (voice, contains
`aliced`), `glagold`, `mediad`, `updatesd`, `wifid`, `telemetry`, `calld`, plus
`fluent-bit` for logs. The launcher restarts a role that dies; several deaths in
a row put it into a quarantine of about ten minutes.

**Linkplay** (`linkplay.service` → `/usr/bin/linkplay.sh`): sets the codec
routing with `amixer`, then starts `rootApp`, which runs `appmainprog`,
`btservice` and the Harman DSP pieces. Bluetooth playback belongs here, not to
quasar. Restarting `linkplay.service` plays the JBL start-up chime.

## Local ports

| Port | Owner | Use |
|---|---|---|
| 9875 | `aliced` (in `maind --yiod`) | Alice state events and commands, see [06](06-quasar-ipc.md) |
| 9906 | quasar `audioclient` | `MediaRequest` playback of URLs |
| 9899 | `glagold` | glagol internals |
| 9891 | `external_commandsd` | external commands |
| 1961 | `glagold` | glagol WebSocket for apps on the LAN (needs a cloud-issued token) |
| 2222 | Dropbear (ours) | SSH |
| 8008, 24721, 18883 | ytm-alice (ours) | DIAL cast target, web panel, local audio relay |

## Logs

- `/data/quasar/launcher2.log`: when each quasar role started.
- `/data/quasar/daemons_logs/<role>.log` and `.log.1`: about 100 KB each, so they
  rotate within minutes. `yiod.log` is the most useful one: it shows recognition,
  VINS requests and every directive (`play_tts`, `cancelDialog` and so on).
- `journalctl` is empty: the journal is not kept on this firmware.
