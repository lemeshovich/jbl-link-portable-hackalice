# jbl-link-portable-hackalice

[English](README.md) · **Українська** · [Русский](README.ru.md)

Root-доступ, захист від OTA, **YouTube Music голосом** і **відповіді LLM
власним голосом Аліси** на JBL Link Portable з Яндекс Алісою, на оригінальній
прошивці 2021 року.

> Це незалежний проєкт. Він спирається на вікі
> [hackalice / Yandex Station Extended](https://hackalice.me/) і на
> [pull request #141](https://github.com/Marker284/yandex-station-extended/pull/141)
> від lonsdaleite, але не є частиною жодного з них. Див. [Подяки](#подяки).

## Що працює

На колонці, яку просто ввімкнули, без жодних команд у терміналі:

- **«Алиса, включи на ютубе Daft Punk»** знаходить музику в YouTube Music і
  вмикає її. Аліса каже «Включаю» своїм голосом; її звичне «это устройство не
  может воспроизводить видео» чи «нужна подписка Плюс» не звучить, навіть
  першим складом.
- **«пауза», «продолжи», «дальше», «выключи музыку», «что сейчас играет»**
  працюють з цією музикою. Повторне «дальше» щоразу знову перемикає трек.
- Власна фраза, наприклад голе **«Алиса, включи музыку»**, може вмикати ваші
  лайки з YouTube Music.
- Звук іде через DSP Harman колонки, тим самим шляхом, що й Bluetooth, і кнопки
  гучності працюють.
- **«Алиса, спроси нейросеть, …»**, питання, що починаються з «объясни»,
  «почему» тощо, і все, на що сама Аліса відповісти не може, ідуть до Gemini
  (або будь-якої OpenAI-сумісної моделі), а Аліса читає відповідь своїм
  голосом. Розмова продовжується без повторного слова активації. Див.
  [llm-alice](docs/uk/15-llm-alice.md).
- Root SSH (лише за ключем), автоматичний запуск усього після ввімкнення і
  заблоковані в три шари оновлення OTA.
- Відтворення через Bluetooth і сама Аліса працюють як раніше.

## Пристрій

| | |
|---|---|
| Колонка | JBL Link Portable, версія з Яндекс Алісою |
| Прошивка | `1.80.0.16.1086317237.20210928`, quasar 3.21 |
| SoC | MediaTek MT8167, userspace ARMv7 |
| ОС | Yocto, systemd, ядро 4.4.22, BusyBox 1.24.1 |

Новіші прошивки відрізняються в суттєвих тут місцях (у їхньому квазарі є IPC
`AliceCapability`, якого в цій немає). Зміни написані так, щоб розпізнавати
стару прошивку й не чіпати новіші, але перевірені лише на версії вище.

## Спершу прочитайте

- Колонку можна зробити непрацездатною. Усе тут зроблено на одному пристрої й
  опубліковано як нотатки, а не як підтримуваний продукт.
- YouTube відмовляється віддавати потік на адреси хмарних провайдерів. Якщо
  колонка виходить в інтернет через VPN чи проксі в дата-центрі, пошук
  працюватиме, а відтворення — ні. Див. [Мережа](docs/uk/12-network.md).
- Таке використання YouTube може порушувати його умови.

## Документація

1. [Пристрій і прошивка](docs/uk/01-device-and-firmware.md)
2. [Root і SSH](docs/uk/02-root-and-ssh.md)
3. [Збереження змін і автозапуск](docs/uk/03-persistence-and-autostart.md)
4. [Захист від OTA](docs/uk/04-ota-protection.md)
5. [Перевизначення прошивки](docs/uk/05-overriding-the-firmware.md)
6. [IPC квазара](docs/uk/06-quasar-ipc.md)
7. [Звук](docs/uk/07-audio.md)
8. [ytm-play, AAC-плеєр](docs/uk/08-ytm-play.md)
9. [Порт ytm-alice](docs/uk/09-ytm-alice-port.md)
10. [Голосове керування на старій прошивці](docs/uk/10-voice-control.md)
11. [Quasar Bridge (і чому він не підходить)](docs/uk/11-quasar-bridge.md)
12. [Мережа](docs/uk/12-network.md)
13. [Усунення несправностей](docs/uk/13-troubleshooting.md)
14. [Нотатки з дослідження заліза](docs/uk/14-hardware-research.md)
15. [llm-alice: відповіді LLM голосом Аліси](docs/uk/15-llm-alice.md)

## Структура репозиторію

| Шлях | Що |
|---|---|
| `patches/` | Патч до [ytm-alice](https://github.com/SPAWNRYS-ban/ytm-alice) для цієї прошивки і скрипт, що збирає з ним `ytm-alice` та `ytm-alice-resolver` |
| `ytm-play/` | Наш AAC/MP4-плеєр із власним faad2 (GPL-2.0) |
| `llm-alice/` | Демон для LLM: джерела, тести, скрипт збірки |
| `tools/` | Невеликі утиліти для IPC та ALSA: `alicesay`, `alicedump`, `qcancel`, `ipcclean`, `qplay` |
| `device/` | Файли в тому вигляді, як вони встановлені на колонці: фреймворк автозапуску й хуки, конфіг ALSA, обгортка плеєра, збірка Dropbear |
| `docs/` | Усе, що з'ясувалось по дорозі, англійською, українською й російською |

## Відтворювані збірки

Кожен бінарник у релізі, крім Dropbear, збирається з цього репозиторію побайтово
ідентичним за допомогою скриптів поруч із джерелами. SHA-256 кожного є в нотатках
до релізу та в `SHA256SUMS`. Dropbear зібрано з того самого коміту й з тими
самими опціями в ітеративному сеансі, і точно він не відтворюється; про це
сказано в його скрипті збірки.

## Швидкий старт

Порядок важливий: кожен крок спирається на попередній.

1. Отримати root: [docs/02](docs/uk/02-root-and-ssh.md).
2. Встановити фреймворк автозапуску й SSH: [docs/03](docs/uk/03-persistence-and-autostart.md).
3. Заблокувати OTA: [docs/04](docs/uk/04-ota-protection.md).
4. Скопіювати бінарники в `/data/ytm/bin/`, конфіг ALSA — у `/data/ytm/`, а хуки
   `05-no-ota.sh` і `30-ytm.sh` — у `/data/jbl-autostart/hooks.d/`:
   [docs/09](docs/uk/09-ytm-alice-port.md).
5. Один раз увійти в YouTube командою `ytm-alice login tv`.
6. За бажанням, для LLM: покласти API-ключ у `/data/llm/key`, бінарник — у
   `/data/llm/`, а хук `40-llm.sh` — до решти:
   [docs/15](docs/uk/15-llm-alice.md#встановлення). Потім перезавантажити.

## Подяки

- [lonsdaleite](https://github.com/lonsdaleite), PR #141 до Yandex Station
  Extended: маркери ADB, плагін і фреймворк автозапуску через Fluent Bit, а також
  скрипт служби SSH, з яких починається все тут. CC BY 4.0, див.
  [device/jbl-autostart/NOTICE.md](device/jbl-autostart/NOTICE.md).
- [Yandex Station Extended / hackalice.me](https://hackalice.me/) від Марка
  Писаренка: вікі, з якої виросла ця робота.
- [ytm-alice](https://github.com/SPAWNRYS-ban/ytm-alice): клієнт YouTube Music і
  голосова інтеграція, які розширює патч.
- [Quasar Bridge](https://github.com/skrashevich/quasar-bridge) від skrashevich:
  розглянутий у [docs/11](docs/uk/11-quasar-bridge.md).
- [AlexxIT/YandexStation](https://github.com/AlexxIT/YandexStation): його запити
  синтезу мовлення показали, як змусити Алісу говорити локально, і що сценарій
  фраз розумного дому не фільтрується.
- [faad2](https://github.com/knik0/faad2), [QuickJS-ng](https://github.com/quickjs-ng/quickjs),
  [Dropbear](https://github.com/mkj/dropbear), [Zig](https://ziglang.org/).
- [quasar-protos](https://github.com/v3rt3p/quasar-protos): публічні визначення
  protobuf, взяті за відправну точку.

## Ліцензія

MIT, див. [LICENSE](LICENSE), за винятком:

- `ytm-play/`: GPL-2.0, бо містить faad2, див. [ytm-play/LICENSE](ytm-play/LICENSE)
- `device/jbl-autostart/`: файли, скопійовані з PR #141, лишаються під CC BY 4.0
- `patches/ytm-alice-quasar-3.21.patch`: MIT, як і сам ytm-alice
