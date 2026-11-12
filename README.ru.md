# jbl-link-portable-hackalice

[English](README.md) · [Українська](README.uk.md) · **Русский**

Root-доступ, защита от OTA, **YouTube Music голосом** и **ответы LLM
собственным голосом Алисы** на JBL Link Portable с Яндекс Алисой, на оригинальной
прошивке 2021 года.

> Это независимый проект. Он опирается на вики
> [hackalice / Yandex Station Extended](https://hackalice.me/) и на
> [pull request #141](https://github.com/Marker284/yandex-station-extended/pull/141)
> от lonsdaleite, но не является частью ни одного из них. См. [Благодарности](#благодарности).

## Что работает

На колонке, которую просто включили, без каких-либо команд в терминале:

- **«Алиса, включи на ютубе Daft Punk»** находит музыку в YouTube Music и
  включает её. Алиса говорит «Включаю» своим голосом; её привычное «это
  устройство не может воспроизводить видео» или «нужна подписка Плюс» не звучит,
  даже первым слогом.
- **«пауза», «продолжи», «дальше», «выключи музыку», «что сейчас играет»**
  работают с этой музыкой. Повторное «дальше» каждый раз снова переключает трек.
- Своя фраза, например просто **«Алиса, включи музыку»**, может включать ваши
  лайки из YouTube Music.
- Звук идёт через DSP Harman колонки, тем же путём, что и Bluetooth, и кнопки
  громкости работают.
- **«Алиса, спроси нейросеть, …»**, вопросы, начинающиеся с «объясни», «почему»
  и т. п., и всё, на что сама Алиса ответить не может, уходят в Gemini (или любую
  OpenAI-совместимую модель), а Алиса читает ответ своим голосом. Разговор
  продолжается без повторного слова активации. См.
  [llm-alice](docs/ru/15-llm-alice.md).
- Root SSH (только по ключу), автоматический запуск всего после включения и
  заблокированные в три слоя обновления OTA.
- Воспроизведение по Bluetooth и сама Алиса работают как раньше.

## Устройство

| | |
|---|---|
| Колонка | JBL Link Portable, версия с Яндекс Алисой |
| Прошивка | `1.80.0.16.1086317237.20210928`, quasar 3.21 |
| SoC | MediaTek MT8167, userspace ARMv7 |
| ОС | Yocto, systemd, ядро 4.4.22, BusyBox 1.24.1 |

Более новые прошивки отличаются в важных здесь местах (в их квазаре есть IPC
`AliceCapability`, которого в этой нет). Изменения написаны так, чтобы
распознавать старую прошивку и не трогать новые, но проверены только на версии
выше.

## Сначала прочитайте

- Колонку можно сделать неработоспособной. Всё здесь сделано на одном устройстве
  и опубликовано как заметки, а не как поддерживаемый продукт.
- YouTube отказывается отдавать поток на адреса облачных провайдеров. Если
  колонка выходит в интернет через VPN или прокси в дата-центре, поиск будет
  работать, а воспроизведение — нет. См. [Сеть](docs/ru/12-network.md).
- Такое использование YouTube может нарушать его условия.

## Документация

1. [Устройство и прошивка](docs/ru/01-device-and-firmware.md)
2. [Root и SSH](docs/ru/02-root-and-ssh.md)
3. [Сохранение изменений и автозапуск](docs/ru/03-persistence-and-autostart.md)
4. [Защита от OTA](docs/ru/04-ota-protection.md)
5. [Переопределение прошивки](docs/ru/05-overriding-the-firmware.md)
6. [IPC квазара](docs/ru/06-quasar-ipc.md)
7. [Звук](docs/ru/07-audio.md)
8. [ytm-play, AAC-плеер](docs/ru/08-ytm-play.md)
9. [Порт ytm-alice](docs/ru/09-ytm-alice-port.md)
10. [Голосовое управление на старой прошивке](docs/ru/10-voice-control.md)
11. [Quasar Bridge (и почему он не подходит)](docs/ru/11-quasar-bridge.md)
12. [Сеть](docs/ru/12-network.md)
13. [Устранение неполадок](docs/ru/13-troubleshooting.md)
14. [Заметки по исследованию железа](docs/ru/14-hardware-research.md)
15. [llm-alice: ответы LLM голосом Алисы](docs/ru/15-llm-alice.md)

## Структура репозитория

| Путь | Что |
|---|---|
| `patches/` | Патч к [ytm-alice](https://github.com/SPAWNRYS-ban/ytm-alice) для этой прошивки и скрипт, который собирает с ним `ytm-alice` и `ytm-alice-resolver` |
| `ytm-play/` | Наш AAC/MP4-плеер с собственным faad2 (GPL-2.0) |
| `llm-alice/` | Демон для LLM: исходники, тесты, скрипт сборки |
| `tools/` | Небольшие утилиты для IPC и ALSA: `alicesay`, `alicedump`, `qcancel`, `ipcclean`, `qplay` |
| `device/` | Файлы в том виде, в каком они установлены на колонке: фреймворк автозапуска и хуки, конфиг ALSA, обёртка плеера, сборка Dropbear |
| `docs/` | Всё, что выяснилось по пути, на английском, украинском и русском |

## Воспроизводимые сборки

Каждый бинарник в релизе, кроме Dropbear, собирается из этого репозитория
побайтово идентичным с помощью скриптов рядом с исходниками. SHA-256 каждого
есть в заметках к релизу и в `SHA256SUMS`. Dropbear собран из того же коммита и
с теми же опциями в итеративном сеансе и точно не воспроизводится; об этом
сказано в его скрипте сборки.

## Быстрый старт

Порядок важен: каждый шаг опирается на предыдущий.

1. Получить root: [docs/02](docs/ru/02-root-and-ssh.md).
2. Установить фреймворк автозапуска и SSH: [docs/03](docs/ru/03-persistence-and-autostart.md).
3. Заблокировать OTA: [docs/04](docs/ru/04-ota-protection.md).
4. Скопировать бинарники в `/data/ytm/bin/`, конфиг ALSA — в `/data/ytm/`, а хуки
   `05-no-ota.sh` и `30-ytm.sh` — в `/data/jbl-autostart/hooks.d/`:
   [docs/09](docs/ru/09-ytm-alice-port.md).
5. Один раз войти в YouTube командой `ytm-alice login tv`.
6. По желанию, для LLM: положить API-ключ в `/data/llm/key`, бинарник — в
   `/data/llm/`, а хук `40-llm.sh` — к остальным:
   [docs/15](docs/ru/15-llm-alice.md#установка). Затем перезагрузить.

## Благодарности

- [lonsdaleite](https://github.com/lonsdaleite), PR #141 в Yandex Station
  Extended: маркеры ADB, плагин и фреймворк автозапуска через Fluent Bit, а также
  скрипт службы SSH, с которых начинается всё здесь. CC BY 4.0, см.
  [device/jbl-autostart/NOTICE.md](device/jbl-autostart/NOTICE.md).
- [Yandex Station Extended / hackalice.me](https://hackalice.me/) от Марка
  Писаренко: вики, из которой выросла эта работа.
- [ytm-alice](https://github.com/SPAWNRYS-ban/ytm-alice): клиент YouTube Music и
  голосовая интеграция, которые расширяет патч.
- [Quasar Bridge](https://github.com/skrashevich/quasar-bridge) от skrashevich:
  рассмотрен в [docs/11](docs/ru/11-quasar-bridge.md).
- [AlexxIT/YandexStation](https://github.com/AlexxIT/YandexStation): его запросы
  синтеза речи показали, как заставить Алису говорить локально, и что сценарий
  фраз умного дома не фильтруется.
- [faad2](https://github.com/knik0/faad2), [QuickJS-ng](https://github.com/quickjs-ng/quickjs),
  [Dropbear](https://github.com/mkj/dropbear), [Zig](https://ziglang.org/).
- [quasar-protos](https://github.com/v3rt3p/quasar-protos): публичные определения
  protobuf, взятые за отправную точку.

## Лицензия

MIT, см. [LICENSE](LICENSE), за исключением:

- `ytm-play/`: GPL-2.0, так как содержит faad2, см. [ytm-play/LICENSE](ytm-play/LICENSE)
- `device/jbl-autostart/`: файлы, скопированные из PR #141, остаются под CC BY 4.0
- `patches/ytm-alice-quasar-3.21.patch`: MIT, как и сам ytm-alice
