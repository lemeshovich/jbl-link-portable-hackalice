# Troubleshooting

**English** · [Українська](../uk/12-troubleshooting.md) · [Русский](../ru/12-troubleshooting.md)

## No sound at all, Alice silent too

Most often the codec gain was dropped ([07](07-audio.md#the-boot-time-gain-bug)).

```
amixer -c 0 cget numid=7 | grep "  : values="      # 524288 is normal, 64 is the problem
/data/ytm/bin/ipcclean dmix
systemctl restart linkplay.service                  # the JBL chime plays
```

If it happens after every boot, something still uses `amixer sget` or `sset` on
the simple `Master` element.

## Bluetooth silent, Alice fine

Stale `dmix` attachments. Same two commands as above.

## Alice "thinks" forever

LEDs spinning, the wake word is ignored, music ducked. A reply was dropped and the
dialog waits for it:

```
/data/ytm/bin/alicesay ""
```

## Alice listens forever

LEDs on as when listening:

```
/data/ytm/bin/alicesay ""; sleep 2; /data/ytm/bin/qcancel -f 170 -E
```

If it is still stuck, restart only the voice role:

```
kill $(ps | grep "[m]aind --yiod" | awk '{print $1}')
```

`quasar_launcher2` brings it back within seconds. Do not kill it repeatedly: a few
deaths in a row put it into a quarantine of about ten minutes.

## Voice commands behave oddly

Read the journal first; it shows every state change and every action:

```
cat /tmp/ytm-voice.log
grep -E "wake word|intercepted|announce|speech sent|not picked up|restored" /tmp/ytm-voice.log
```

Then `yiod.log` around the same time, before it rotates (minutes):

```
grep -a "HH:MM:S" /data/quasar/daemons_logs/yiod.log*
```

## ytm-alice asks to log in after a reboot

`droproot_setup` reset the session file to a mode other than 600. The unit from
`30-ytm.sh` fixes that before every start; check that the hook is installed.

## Copying commands into PuTTY

- Cyrillic text pasted into PuTTY can arrive broken, and some characters trigger
  shell completion. Keep commands ASCII; `alicesay` has "Включаю" built in for
  this reason.
- Long commands pasted twice in a row get glued together. Check the prompt before
  pasting the next one.

## llm-alice

- Its journal is `/tmp/llm-alice.log`: every routed phrase, every answer, which
  model produced it and which speech scenario read it.
- `llm-alice ask` without arguments checks the key and the network without any
  voice involved.
- "Нейросеть сейчас не отвечает" after a few seconds means the model failed even
  after the retries; the reason is in the journal (often HTTP 503, the model
  being overloaded).
- See [14](14-llm-alice.md) for how phrases are routed.

## Disabling things

```
chattr -R -i /data/jbl-autostart
touch /data/jbl-autostart/ytm-off       # ytm-alice no longer starts at boot
touch /data/jbl-autostart/llm-off       # llm-alice no longer starts at boot
chattr -R +i /data/jbl-autostart
```

Stop them now: `systemctl stop ytm-alice.service` or `systemctl stop llm-alice.service`.
