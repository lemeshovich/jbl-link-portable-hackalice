# OTA protection

**English** · [Українська](../uk/04-ota-protection.md) · [Русский](../ru/04-ota-protection.md)

An update would replace the firmware this whole project depends on, and could
close the ADB window. Three layers keep it out.

| Layer | How | Persistent |
|---|---|---|
| 1 | `chattr +i /bkupgrade` | yes, an attribute on UBIFS |
| 2 | `/bin/true` bind-mounted over `/system/vendor/quasar/device_ota_update.sh` | re-applied at boot by `05-no-ota.sh` |
| 3 | `/bin/true` bind-mounted over `/system/vendor/quasar/a01localupdate` | re-applied at boot by `05-no-ota.sh` |

`/bkupgrade` is where update images are downloaded. Make it immutable **without**
`-R` and only when it holds no downloaded image (`ls -la /bkupgrade`), as the
hackalice guide explains.

The hook ([device/jbl-autostart/hooks.d/05-no-ota.sh](../../device/jbl-autostart/hooks.d/05-no-ota.sh))
mounts only if the target exists and is not mounted yet, so running it twice is
harmless.

Check after a reboot:

```
mount | grep -c "device_ota\|a01local"     # 2
touch /bkupgrade/test                       # must fail: Operation not permitted
```

Undo:

```
chattr -i /bkupgrade
chattr -R -i /data/jbl-autostart
rm /data/jbl-autostart/hooks.d/05-no-ota.sh
chattr -R +i /data/jbl-autostart
reboot
```
