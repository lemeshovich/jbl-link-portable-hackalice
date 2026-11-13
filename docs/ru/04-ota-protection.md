# Защита от обновлений

[English](../en/04-ota-protection.md) · [Українська](../uk/04-ota-protection.md) · **Русский**

Обновление заменило бы прошивку, на которой держится весь проект, и могло бы
закрыть окно ADB. Поэтому блокируем его сразу в трёх местах.

| № | Что делаем | Переживёт ли перезагрузку |
|---|---|---|
| 1 | `chattr +i /bkupgrade` | да, это атрибут на UBIFS |
| 2 | `/bin/true` монтируется поверх `/system/vendor/quasar/device_ota_update.sh` | да, хук `05-no-ota.sh` делает это при каждой загрузке |
| 3 | `/bin/true` монтируется поверх `/system/vendor/quasar/a01localupdate` | да, тем же хуком |

В `/bkupgrade` скачиваются образы обновлений. Делайте его неизменяемым **без**
`-R` и только тогда, когда там нет уже скачанного образа (`ls -la /bkupgrade`) —
так советует и инструкция hackalice.

Хук ([device/jbl-autostart/hooks.d/05-no-ota.sh](../../device/jbl-autostart/hooks.d/05-no-ota.sh))
монтирует только тогда, когда файл существует и ещё не смонтирован, так что
повторный запуск ничего не сломает.

## Проверка

После перезагрузки:

```
mount | grep -c "device_ota\|a01local"     # должно быть 2
touch /bkupgrade/test                       # должно отказать: Operation not permitted
```

## Как отменить

```
chattr -i /bkupgrade
chattr -R -i /data/jbl-autostart
rm /data/jbl-autostart/hooks.d/05-no-ota.sh
chattr -R +i /data/jbl-autostart
reboot
```
