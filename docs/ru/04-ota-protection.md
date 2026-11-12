# Защита от OTA

[English](../en/04-ota-protection.md) · [Українська](../uk/04-ota-protection.md) · **Русский**

Обновление заменило бы прошивку, от которой зависит весь этот проект, и могло бы
закрыть окно ADB. Его не пускают три слоя защиты.

| Слой | Как | Сохраняется |
|---|---|---|
| 1 | `chattr +i /bkupgrade` | да, это атрибут на UBIFS |
| 2 | `/bin/true`, смонтированный (bind) поверх `/system/vendor/quasar/device_ota_update.sh` | восстанавливается при загрузке хуком `05-no-ota.sh` |
| 3 | `/bin/true`, смонтированный (bind) поверх `/system/vendor/quasar/a01localupdate` | восстанавливается при загрузке хуком `05-no-ota.sh` |

`/bkupgrade` — это место, куда скачиваются образы обновлений. Делайте его
неизменяемым **без** `-R` и только тогда, когда в нём нет скачанного образа
(`ls -la /bkupgrade`), как объясняет инструкция hackalice.

Хук ([device/jbl-autostart/hooks.d/05-no-ota.sh](../../device/jbl-autostart/hooks.d/05-no-ota.sh))
монтирует, только если цель существует и ещё не смонтирована, поэтому повторный
запуск безвреден.

Проверка после перезагрузки:

```
mount | grep -c "device_ota\|a01local"     # 2
touch /bkupgrade/test                       # должно завершиться ошибкой: Operation not permitted
```

Отмена:

```
chattr -i /bkupgrade
chattr -R -i /data/jbl-autostart
rm /data/jbl-autostart/hooks.d/05-no-ota.sh
chattr -R +i /data/jbl-autostart
reboot
```
