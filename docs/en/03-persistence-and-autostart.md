# Persistence and autostart

**English** · [Українська](../uk/03-persistence-and-autostart.md) · [Русский](../ru/03-persistence-and-autostart.md)

Nothing in the read-only root filesystem can be changed, and `droproot_setup`
resets `/data` permissions at every boot. Autostart therefore has to hang on
something the firmware already starts and that reads configuration from `/data`.

## The Fluent Bit plugin (PR #141)

The idea and the code are lonsdaleite's
([PR #141](https://github.com/Marker284/yandex-station-extended/pull/141)):

1. The vendor `fluent-bit` 1.4.6 reads `/data/quasar/fluent-bit/device.cfg`.
   One line is appended to it: `@INCLUDE /data/jbl-autostart/fluent-service.cfg`,
   and the file is protected with `chattr +i` so it is not regenerated.
2. That include loads a tiny input plugin,
   [`flb-in_jbl_bootstrap.so`](../../device/jbl-autostart/fluent_bootstrap.c), whose
   only job is to run `/data/jbl-autostart/bootstrap.sh` once and return.
3. `bootstrap.sh` installs `jbl-autostart.service` into `/run/systemd/system`
   and starts it without waiting.
4. The service runs `run.sh`, which executes `hooks.d/*.sh` in order, after
   checking ownership and permissions of each hook.

The whole `/data/jbl-autostart` tree is `chattr -R +i`. To change a hook:

```
chattr -R -i /data/jbl-autostart
# edit, chmod 700, chown 0:0
chattr -R +i /data/jbl-autostart
```

The PR builds the plugin with clang and lld; here it is built with Zig, giving the
same file on every run: [device/jbl-autostart/build-plugin.sh](../../device/jbl-autostart/build-plugin.sh).

## Our hooks

| Hook | What |
|---|---|
| `05-no-ota.sh` | Re-applies the OTA protection at every boot, see [04](04-ota-protection.md) |
| `10-ssh.sh` | From the PR: starts Dropbear |
| `30-ytm.sh` | Installs `ytm-alice.service` into `/run` and starts it without waiting |
| `40-llm.sh` | Installs `llm-alice.service` (`Restart=on-failure`) the same way, see [15](15-llm-alice.md) |
| `90-marker.sh` | From the PR: writes the boot id, proving that the hooks ran |

Creating `/data/jbl-autostart/ytm-off` makes `30-ytm.sh` skip ytm-alice, and
`/data/jbl-autostart/llm-off` does the same for llm-alice.

## Hooks must return quickly

Hooks run inside one oneshot service with a 30 second limit. Anything long-lived
belongs in its own unit. `30-ytm.sh` only writes the unit and calls
`systemctl start --no-block`; ordering after `linkplay.service` and the network is
left to systemd.

A background process started from a hook or from an `ExecStartPre` line is killed
by systemd once that step ends. Run long-lived things as the main process of a
unit instead.

## The boot-time silence that was not about autostart

For a long time ytm-alice started at boot left the speaker silent, while the same
binary started over SSH later was fine. Delays, waiting for the audio stack,
warming up the mixer, systemd ordering and the process environment all made no
difference. The real cause was a single `amixer sget Master` in ytm-alice, which
only on a freshly booted speaker drops the codec gain to almost zero. All manual
tests had been done right after restarting `linkplay.service`, which happens to
undo it. See [07, the boot-time gain bug](07-audio.md#the-boot-time-gain-bug).
