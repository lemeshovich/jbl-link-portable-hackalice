# jbl-link-portable-hackalice

Root access, OTA protection and **YouTube Music by voice** on a JBL Link Portable
with Yandex Alice, running the original 2021 firmware.

> This is an independent project. It builds on the
> [hackalice / Yandex Station Extended](https://hackalice.me/) wiki and on
> [pull request #141](https://github.com/Marker284/yandex-station-extended/pull/141)
> by lonsdaleite, but is not part of either. See [Credits](#credits).

## What works

On a speaker that has been switched on normally, with nothing typed anywhere:

- **"Алиса, включи на ютубе Daft Punk"** finds the music on YouTube Music and plays
  it. Alice says "Включаю" in her own voice; her usual "this device cannot play
  video" or "you need a Plus subscription" is never heard, not even its first
  syllable.
- **"пауза", "продолжи", "дальше", "выключи музыку", "что сейчас играет"** work on
  that music. Repeating "дальше" skips again every time.
- Sound goes through the speaker's Harman DSP, the same path Bluetooth uses, and
  the volume buttons work.
- Root SSH (key only), automatic start of everything after power-on, and OTA
  updates blocked in three layers.
- Bluetooth playback and Alice herself keep working as before.

## Device

| | |
|---|---|
| Speaker | JBL Link Portable, Yandex Alice edition |
| Firmware | `1.80.0.16.1086317237.20210928`, quasar 3.21 |
| SoC | MediaTek MT8167, ARMv7 userspace |
| OS | Yocto, systemd, kernel 4.4.22, BusyBox 1.24.1 |

Newer firmware differs in places that matter here (its quasar has the
`AliceCapability` IPC that this one lacks). The changes are written to detect the
old firmware and leave newer ones alone, but they have only been tested on the
version above.

## Read first

- You can make your speaker unusable. Everything here was done on one device and
  is published as notes, not as a supported product.
- YouTube refuses to stream to addresses of cloud providers. If the speaker
  reaches the internet through a VPN or proxy in a data centre, search works but
  playback does not. See [Network](docs/12-network.md).
- Using YouTube this way may break its terms of service.

## Documentation

1. [Device and firmware](docs/01-device-and-firmware.md)
2. [Root and SSH](docs/02-root-and-ssh.md)
3. [Persistence and autostart](docs/03-persistence-and-autostart.md)
4. [OTA protection](docs/04-ota-protection.md)
5. [Overriding the firmware](docs/05-overriding-the-firmware.md)
6. [Quasar IPC](docs/06-quasar-ipc.md)
7. [Audio](docs/07-audio.md)
8. [ytm-play, the AAC player](docs/08-ytm-play.md)
9. [ytm-alice port](docs/09-ytm-alice-port.md)
10. [Voice control on legacy firmware](docs/10-voice-control.md)
11. [Quasar Bridge (and why it does not fit)](docs/11-quasar-bridge.md)
12. [Network](docs/12-network.md)
13. [Troubleshooting](docs/13-troubleshooting.md)
14. [Hardware research notes](docs/14-hardware-research.md)

## Repository layout

| Path | What |
|---|---|
| `patches/` | The patch to [ytm-alice](https://github.com/SPAWNRYS-ban/ytm-alice) for this firmware, and a script that builds `ytm-alice` and `ytm-alice-resolver` with it |
| `ytm-play/` | Our AAC/MP4 player with its own faad2 (GPL-2.0) |
| `tools/` | Small IPC and ALSA tools: `alicesay`, `alicedump`, `qcancel`, `ipcclean`, `qplay` |
| `device/` | Files as installed on the speaker: autostart framework and hooks, ALSA config, player wrapper, Dropbear build |
| `docs/` | Everything learned on the way |

## Reproducible builds

Every binary in the release except Dropbear is built byte-for-byte identical from
this repository with the scripts next to its sources. The SHA-256 of each is in
the release notes and in `SHA256SUMS`. Dropbear was built from the same commit and
options in an iterative session and is not reproduced exactly; its build script
says so.

## Quick start

The order matters, because each step relies on the previous one.

1. Get root: [docs/02](docs/02-root-and-ssh.md).
2. Install the autostart framework and SSH: [docs/03](docs/03-persistence-and-autostart.md).
3. Block OTA: [docs/04](docs/04-ota-protection.md).
4. Copy the binaries to `/data/ytm/bin/`, the ALSA config to `/data/ytm/`, and the
   `05-no-ota.sh` and `30-ytm.sh` hooks into `/data/jbl-autostart/hooks.d/`:
   [docs/09](docs/09-ytm-alice-port.md).
5. Log in to YouTube once with `ytm-alice login tv`, then reboot.

## Credits

- [lonsdaleite](https://github.com/lonsdaleite), PR #141 to Yandex Station
  Extended: the ADB markers, the Fluent Bit autostart plugin and framework, and
  the SSH service script that everything here starts from. CC BY 4.0, see
  [device/jbl-autostart/NOTICE.md](device/jbl-autostart/NOTICE.md).
- [Yandex Station Extended / hackalice.me](https://hackalice.me/) by Mark
  Pisarenko: the wiki this work grew from.
- [ytm-alice](https://github.com/SPAWNRYS-ban/ytm-alice): the YouTube Music
  client and voice integration that the patch extends.
- [Quasar Bridge](https://github.com/skrashevich/quasar-bridge) by skrashevich:
  evaluated in [docs/11](docs/11-quasar-bridge.md).
- [AlexxIT/YandexStation](https://github.com/AlexxIT/YandexStation): its
  text-to-speech request pointed to the way Alice can be made to speak locally.
- [faad2](https://github.com/knik0/faad2), [QuickJS-ng](https://github.com/quickjs-ng/quickjs),
  [Dropbear](https://github.com/mkj/dropbear), [Zig](https://ziglang.org/).
- [quasar-protos](https://github.com/v3rt3p/quasar-protos): public protobuf
  definitions used as a starting point.

## License

MIT, see [LICENSE](LICENSE), except:

- `ytm-play/`: GPL-2.0, because it includes faad2, see [ytm-play/LICENSE](ytm-play/LICENSE)
- `device/jbl-autostart/`: the files copied from PR #141 stay under CC BY 4.0
- `patches/ytm-alice-quasar-3.21.patch`: MIT, like ytm-alice itself
