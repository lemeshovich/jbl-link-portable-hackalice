# jbl-link-portable-hackalice

[English](README.md) · **Українська** · [Русский](README.ru.md)

Root-доступ, захист від оновлень, музика з YouTube Music на голосові команди і
відповіді нейромережі голосом самої Аліси — на колонці JBL Link Portable з
Яндекс Алісою та заводською прошивкою 2021 року.

> Проєкт самостійний. Він виріс із вікі
> [hackalice / Yandex Station Extended](https://hackalice.me/) і з
> [pull request #141](https://github.com/Marker284/yandex-station-extended/pull/141)
> від lonsdaleite, але до жодного з них не належить. Хто що зробив — у розділі
> [Подяки](#подяки).

## Можливості

Усе працює одразу після ввімкнення колонки, термінал для цього не потрібен.

- **«Алиса, включи на ютубе Daft Punk»** — колонка знаходить трек у YouTube Music
  і вмикає його. Аліса своїм голосом відповідає «Включаю», а її звичні відмовки
  про те, що відео тут не підтримується або що потрібна підписка Плюс, більше не
  звучать узагалі.
- Музикою можна керувати як завжди: **«пауза»**, **«продолжи»**, **«дальше»**,
  **«выключи музыку»**, **«что сейчас играет»**. «Дальше» спрацьовує хоч кілька
  разів поспіль.
- На власну фразу можна повісити свої лайки з YouTube Music — наприклад, щоб
  просте **«Алиса, включи музыку»** вмикало саме їх.
- Звук іде через DSP Harman, як і з Bluetooth, тож баси на місці, а гучність
  регулюється кнопками.
- **«Алиса, спроси нейросеть, …»**, питання на кшталт «объясни…» чи «почему…», а
  також усе, на що Аліса сама відповіді не знає, отримує Gemini або інша
  модель, сумісна з OpenAI. Відповідь Аліса читає своїм голосом, а розмову можна
  продовжити, не кличучи її знову на ім'я. Докладніше — у розділі про
  [llm-alice](docs/uk/14-llm-alice.md).
- Вхід по SSH з правами root (лише за ключем), автозапуск усього потрібного після
  ввімкнення і три незалежні перешкоди для оновлень.
- Bluetooth і сама Аліса працюють, як і раніше.

## Пристрій

| | |
|---|---|
| Колонка | JBL Link Portable з Яндекс Алісою |
| Прошивка | `1.80.0.16.1086317237.20210928`, quasar 3.21 |
| Процесор | MediaTek MT8167, ARMv7 |
| Система | Yocto, systemd, ядро 4.4.22, BusyBox 1.24.1 |

Усе тут перевірено лише на прошивці з таблиці. Новіші прошивки влаштовані
інакше: Алісою на них керують через інший механізм (`AliceCapability`), якого
в цій версії ще немає. Наш патч до ytm-alice сам визначає, на якій прошивці
працює, і на новіших нічого не змінює, тож там ytm-alice поводиться як зазвичай.

## Перш ніж почати

- Колонку цілком реально перетворити на цеглину. Усе описане зроблено на одному
  екземплярі, і це нотатки, а не продукт із підтримкою.
- Таке використання YouTube може суперечити його правилам.

## Документація

1. [Пристрій і прошивка](docs/uk/01-device-and-firmware.md)
2. [Root і SSH](docs/uk/02-root-and-ssh.md)
3. [Як зберегти зміни й налаштувати автозапуск](docs/uk/03-persistence-and-autostart.md)
4. [Захист від оновлень](docs/uk/04-ota-protection.md)
5. [Як підмінити файли прошивки](docs/uk/05-overriding-the-firmware.md)
6. [Обмін повідомленнями з квазаром](docs/uk/06-quasar-ipc.md)
7. [Звук](docs/uk/07-audio.md)
8. [ytm-play — програвач AAC](docs/uk/08-ytm-play.md)
9. [ytm-alice на старій прошивці](docs/uk/09-ytm-alice-port.md)
10. [Голосове керування на старій прошивці](docs/uk/10-voice-control.md)
11. [Quasar Bridge і чому він тут не підходить](docs/uk/11-quasar-bridge.md)
12. [Якщо щось пішло не так](docs/uk/12-troubleshooting.md)
13. [Що відомо про залізо](docs/uk/13-hardware-research.md)
14. [llm-alice: відповіді нейромережі голосом Аліси](docs/uk/14-llm-alice.md)

## Що є в репозиторії

| Каталог | Вміст |
|---|---|
| `patches/` | патч до [ytm-alice](https://github.com/SPAWNRYS-ban/ytm-alice) під цю прошивку і скрипт, який збирає з ним `ytm-alice` та `ytm-alice-resolver` |
| `ytm-play/` | наш програвач AAC/MP4 із вбудованим faad2 (GPL-2.0) |
| `llm-alice/` | демон для нейромережі: код, тести, скрипт збирання |
| `tools/` | дрібні утиліти для IPC та ALSA: `alicesay`, `alicedump`, `qcancel`, `ipcclean`, `qplay` |
| `device/` | файли в тому вигляді, в якому вони лежать на колонці: автозапуск із хуками, налаштування ALSA, обгортка програвача, збирання Dropbear |
| `docs/` | усе, що вдалося з'ясувати, трьома мовами |

## Збирання з вихідного коду

Усі бінарники з релізу, крім Dropbear, можна зібрати самостійно скриптами, що
лежать поруч із кодом, і результат збіжеться з релізним байт у байт. Контрольні
суми є в `SHA256SUMS` та в описі релізу. Dropbear збирали з того самого коміту й
з тими самими параметрами, але вручну і в кілька заходів, тож точної копії не
вийде — про це прямо сказано в його скрипті.

## Встановлення коротко

Кроки йдуть саме в такому порядку, бо кожен наступний спирається на попередній.

1. Отримуємо root: [docs/02](docs/uk/02-root-and-ssh.md).
2. Сетапимо автозапуск і SSH: [docs/03](docs/uk/03-persistence-and-autostart.md).
3. Блокуємо оновлення: [docs/04](docs/uk/04-ota-protection.md).
4. Кладемо бінарники в `/data/ytm/bin/`, налаштування ALSA — у `/data/ytm/`, а
   хуки `05-no-ota.sh` і `30-ytm.sh` — у `/data/jbl-autostart/hooks.d/`:
   [docs/09](docs/uk/09-ytm-alice-port.md).
5. Один раз входимо в YouTube командою `ytm-alice login tv`.
6. Якщо потрібна нейромережа: записуємо API-ключ у `/data/llm/key`, бінарник
   кладемо в `/data/llm/`, а хук `40-llm.sh` — до інших:
   [docs/14](docs/uk/14-llm-alice.md#встановлення). Після цього перезавантажуємо
   колонку.

## Подяки

- [lonsdaleite](https://github.com/lonsdaleite), PR #141 до Yandex Station
  Extended: маркери ADB, автозапуск через плагін Fluent Bit разом з усім, що
  його оточує, і скрипт служби SSH. З цього все й почалося. Ліцензія CC BY 4.0,
  подробиці — у [device/jbl-autostart/NOTICE.md](device/jbl-autostart/NOTICE.md).
- [Yandex Station Extended / hackalice.me](https://hackalice.me/), Марк Писаренко:
  вікі, з якої виріс цей проєкт.
- [ytm-alice](https://github.com/SPAWNRYS-ban/ytm-alice): клієнт YouTube Music з
  голосовим керуванням, який доповнює наш патч.
- [Quasar Bridge](https://github.com/skrashevich/quasar-bridge), skrashevich: ми
  його ґрунтовно спробували, результат — у [docs/11](docs/uk/11-quasar-bridge.md).
- [AlexxIT/YandexStation](https://github.com/AlexxIT/YandexStation): саме з його
  запитів на озвучення стало зрозуміло, як змусити Алісу говорити локально і що
  сценарій фраз розумного дому не фільтрується.
- [faad2](https://github.com/knik0/faad2), [QuickJS-ng](https://github.com/quickjs-ng/quickjs),
  [Dropbear](https://github.com/mkj/dropbear), [Zig](https://ziglang.org/).
- [quasar-protos](https://github.com/v3rt3p/quasar-protos): відкриті описи
  protobuf, з яких почалося знайомство з протоколом.

## Ліцензія

MIT (див. [LICENSE](LICENSE)), окрім:

- `ytm-play/` — GPL-2.0, бо в ньому faad2 ([ytm-play/LICENSE](ytm-play/LICENSE));
- файлів із PR #141 у `device/jbl-autostart/` — вони лишаються під CC BY 4.0;
- `patches/ytm-alice-quasar-3.21.patch` — MIT, як і сам ytm-alice.
