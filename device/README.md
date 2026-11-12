# Files on the speaker

| Repository | On the speaker |
|---|---|
| `jbl-autostart/` | `/data/jbl-autostart/` (protect with `chattr -R +i` afterwards) |
| `jbl-autostart/build-plugin.sh` | builds `flb-in_jbl_bootstrap.so` for the same directory |
| `ytm/my.conf` | `/data/ytm/my.conf` |
| `ytm/ytm-alice-player` | `/data/ytm/bin/ytm-alice-player` |
| `ssh/build-dropbear.sh` | builds `dropbear` and `dropbearkey` for `/data/jbl-ssh/` |

`jbl-autostart/` combines the framework from PR #141 with our hooks; see
[jbl-autostart/NOTICE.md](jbl-autostart/NOTICE.md) and
[docs/03](../docs/en/03-persistence-and-autostart.md).

Hooks must be owned by root with mode 700 or 755, or `run.sh` refuses to run
them.
