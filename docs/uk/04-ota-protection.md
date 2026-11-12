# Захист від OTA

[English](../en/04-ota-protection.md) · **Українська** · [Русский](../ru/04-ota-protection.md)

Оновлення замінило б прошивку, від якої залежить увесь цей проєкт, і могло б
закрити вікно ADB. Його не пускають три шари захисту.

| Шар | Як | Зберігається |
|---|---|---|
| 1 | `chattr +i /bkupgrade` | так, це атрибут на UBIFS |
| 2 | `/bin/true` змонтований (bind) поверх `/system/vendor/quasar/device_ota_update.sh` | відновлюється при завантаженні хуком `05-no-ota.sh` |
| 3 | `/bin/true` змонтований (bind) поверх `/system/vendor/quasar/a01localupdate` | відновлюється при завантаженні хуком `05-no-ota.sh` |

`/bkupgrade` — це місце, куди завантажуються образи оновлень. Робіть його
незмінним **без** `-R` і лише тоді, коли в ньому немає завантаженого образу
(`ls -la /bkupgrade`), як пояснює інструкція hackalice.

Хук ([device/jbl-autostart/hooks.d/05-no-ota.sh](../../device/jbl-autostart/hooks.d/05-no-ota.sh))
монтує лише тоді, коли ціль існує й ще не змонтована, тож повторний запуск
нешкідливий.

Перевірка після перезавантаження:

```
mount | grep -c "device_ota\|a01local"     # 2
touch /bkupgrade/test                       # має завершитися помилкою: Operation not permitted
```

Скасування:

```
chattr -i /bkupgrade
chattr -R -i /data/jbl-autostart
rm /data/jbl-autostart/hooks.d/05-no-ota.sh
chattr -R +i /data/jbl-autostart
reboot
```
