# Voice control on legacy firmware

What happens between "Алиса, включи на ютубе Daft Punk" and the music, on quasar
3.21, and why each step is there. Everything here was found from the voice
journal (`/tmp/ytm-voice.log`) and `yiod.log`, not guessed.

## The problem

Alice sends every command to the cloud. ytm-alice sees the recognised text and
plays the music itself, but the cloud has also answered, and Alice is about to
say "Воспроизведение видео не поддерживается на этом устройстве" or "Для этого
нужна подписка Плюс…". On newer firmware ytm-alice cancels that through the
`AliceCapability` IPC, which 3.21 does not have.

## What did not work

- Cancelling right when the command is recognised: the answer arrives later, so
  there is nothing to cancel yet.
- Cancelling in a loop for a few seconds: it also closes Alice's microphone, so
  the next command is cut off.
- Replacing the answer with an empty "repeat after me" request: it needs a cloud
  round trip of 0.3 to 2 s, during which the first words are spoken.
- Firmware messages that look like a cancel (`restart_vins_dialog`,
  `sound_cancel`, the action button): no effect on 3.21.

## What works

### 1. Recognise the command

The text is in a state 1 frame; the final state 3 comes in its own frame without
text. The text is carried over (10 s window). A repeated phrase counts as a new
command when a wake word event came in between.

### 2. Mute only Alice

At that moment `ShadowMaster` (Alice's softvol) is saved and set to 0. The music
is on `Master` and keeps playing. The saved value is written to
`/tmp/ytm-shadow-saved`; if the daemon dies while Alice is muted, the next start
restores it.

### 3. Drop the answer before it is spoken

The answer shows up in a state 3 frame with its text before Alice starts
speaking:

```
state 3 -> 3 speech="Воспроизведение видео не поддерживается на этом устройстве."
refusal in state 3 ...: cleanup_directive_controller
state 3 -> 0
```

A reply is treated as a refusal if it mentions "подписк", "видео", "не
поддерживается" or "недоступ", and only within 12 s of a command ytm-alice took,
so Alice is not silenced when asked something else.

`cleanup_directive_controller` is the firmware's own dialog cancel. Sometimes the
reply has already reached the audio path and Alice keeps "speaking" into the muted
channel for up to two seconds; the mute is held as long as she is in state 2 with
a refusal, at most 4 s in total.

### 4. Release the dialog

A dropped reply leaves the dialog waiting for playback that never comes: the LEDs
keep spinning ("thinking") and the wake word is ignored. It is released by a new
request. For "включи" that request is the announcement itself; for "пауза",
"дальше" and the rest it is an empty "repeat after me", which Alice answers with
silence.

### 5. Unmute and announce

`ShadowMaster` is restored about 400 ms after the refusal was dropped (or 2.5 s
after the command if no refusal came). Only then is "Включаю" sent, because the
cancel also cancels a request in flight. If Alice does not react to it within
1.5 s (normally she does in about 50 ms), it is sent once more.

### 6. Close the microphone

After any request of ours the firmware opens the microphone for about five
seconds, which also keeps the music ducked. When Alice enters state 1 within 10 s
of our request **without** a wake word event before it, the daemon sends
`cleanup_directive_controller` a second later, and retries while she is still
listening. A session the user starts with "Алиса" always has the wake word event
first and is never touched.

## A typical journal

```
wake word
state 0 -> 1
state 1 -> 3
ShadowMaster 610,610 -> 0 (Alice muted)
intercepted "Включи на YouTube Daft Punk" -> play
state 3 -> 3 speech="Воспроизведение видео не поддерживается на этом устройстве."
refusal in state 3 ...: cleanup_directive_controller
state 3 -> 0
ShadowMaster restored to 610,610
announce queued "Включаю" (attempt 0)
speech sent after 4ms "Включаю" err=<nil>
state 0 -> 3
state 3 -> 2 speech="Окей, повторяю за вами: «Включаю»."
state 2 -> 1
auto-listen opened after our request
auto-listen still open: cleanup_directive_controller
state 1 -> 0
```

## If Alice gets stuck anyway

- **Thinking forever** (spinning LEDs, ignores "Алиса"): `alicesay ""` releases
  the dialog.
- **Listening forever**: `alicesay ""`, then `qcancel -f 170 -E`; if that does not
  help, restart the voice role: `kill $(ps | grep "[m]aind --yiod" | awk '{print $1}')`.
  The launcher brings it back in a few seconds.
