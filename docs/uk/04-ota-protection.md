# Захист від оновлень

[English](../en/04-ota-protection.md) · **Українська** · [Русский](../ru/04-ota-protection.md)

Оновлення замінило б прошивку, на якій тримається весь проєкт, і могло б закрити
вікно ADB. Тому його блокуємо одразу в трьох місцях.

| № | Що робимо | Чи переживе перезавантаження |
|---|---|---|
| 1 | `chattr +i /bkupgrade` | так, це атрибут на UBIFS |
| 2 | `/bin/true` монтується поверх `/system/vendor/quasar/device_ota_update.sh` | так, хук `05-no-ota.sh` робить це при кожному завантаженні |
| 3 | `/bin/true` монтується поверх `/system/vendor/quasar/a01localupdate` | так, через той самий хук |

У `/bkupgrade` завантажуються образи оновлень. Робіть його незмінним **без** `-R`
і лише тоді, коли там немає вже завантаженого образу (`ls -la /bkupgrade`) — так
радить і інструкція hackalice.

Хук ([device/jbl-autostart/hooks.d/05-no-ota.sh](../../device/jbl-autostart/hooks.d/05-no-ota.sh))
монтує лише тоді, коли файл існує і ще не змонтований, тож повторний запуск нічого
не зламає.

## Перевірка

Після перезавантаження:

```
mount | grep -c "device_ota\|a01local"     # має бути 2
touch /bkupgrade/test                       # має відмовити: Operation not permitted
```

## Як скасувати

```
chattr -i /bkupgrade
chattr -R -i /data/jbl-autostart
rm /data/jbl-autostart/hooks.d/05-no-ota.sh
chattr -R +i /data/jbl-autostart
reboot
```
