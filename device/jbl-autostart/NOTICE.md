# Attribution

The autostart framework in this directory comes from
[pull request #141](https://github.com/Marker284/yandex-station-extended/pull/141)
by [lonsdaleite](https://github.com/lonsdaleite) to
[Yandex Station Extended](https://github.com/Marker284/yandex-station-extended)
(hackalice.me), commit `aca21a42cd0b350ca629de06fa84b9a90b13138f`, licensed under
[CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).

Copied unmodified from that commit:

- `fluent_bootstrap.c`
- `bootstrap.sh`
- `run.sh`
- `jbl-autostart.service`
- `hooks.d/10-ssh.sh`
- `hooks.d/90-marker.sh`

Added in this repository (MIT, see the top-level `LICENSE`):

- `build-plugin.sh`: builds the plugin with Zig instead of clang and ELF lld
- `hooks.d/05-no-ota.sh`: OTA protection, re-applied on every boot
- `hooks.d/30-ytm.sh`: installs and starts the ytm-alice systemd unit
- `plugins.conf`, `fluent-service.cfg`, `managed-by-jbl-link-portable`: the
  configuration as installed by hand on the device

PR #141 was tested on firmware `1.80.0.228.3676569159.20251219`. Everything here
runs on `1.80.0.16.1086317237.20210928` (quasar 3.21) as well.
