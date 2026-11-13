# llm-alice: answers from an LLM in Alice's voice

**English** · [Українська](../uk/14-llm-alice.md) · [Русский](../ru/14-llm-alice.md)

`llm-alice` lets Alice answer with a large language model (Gemini by default) on
quasar 3.21, where [Quasar Bridge](11-quasar-bridge.md) cannot work. Source:
[llm-alice/](../../llm-alice/), Go standard library only, about 2 MB on the
speaker.

## What it does

- **"Алиса, спроси нейросеть, …"** (also "спроси у нейросети", "спроси гемини" and
  similar): the question goes to the model, and Alice reads the answer in her own
  voice.
- **Rules**: phrases starting with "объясни", "сравни", "почему", "как работает",
  "в чем разница", "придумай", "сочини" go to the model directly.
- **Fallback**: when Alice cannot answer ("по вашему запросу ничего не нашлось",
  "так просто не объяснить", "я не знаю что ответить") or only quotes a website
  ("Нашла ответ на …"), her answer is dropped before she speaks and the model
  answers instead. Everything she does well (time, weather, timers, music,
  arithmetic) stays hers.
- **Conversation**: the model remembers the last turns for a few minutes. When
  its answer ends in a question, the speaker keeps listening and the reply is
  said **without** the wake word.
- **"Алиса, спроси нейросеть"** with no question, right after Alice answered
  something herself, sends that same question to the model.
- **"Алиса, начни новый разговор"** (or "забудь разговор", "новая тема") forgets
  the conversation.

## How it works

`llm-alice` sits next to Alice, not between her and Yandex. It reads Alice's
state events from `aliced` ([06](06-quasar-ipc.md)), cancels her own answer with
`cleanup_directive_controller` when needed, mutes only her voice
(`ShadowMaster`, see [07](07-audio.md)) and makes her speak through a
`vins_server_action`. Nothing in the firmware configuration is changed: no
`uniProxyUrl`, no `user_config.dat`. If the daemon stops, Alice is exactly as
before.

### Routing a phrase

The recognised text arrives in a state 1 frame, the final state 3 in the next
one. A leading "Алиса" inside the text is removed: when the microphone is
already open the wake word is recognised as text and no spotter event comes.
Then, in this order:

1. a reset phrase: forget the conversation;
2. a trigger phrase: ask the model; a bare trigger re-asks the last question
   Alice answered herself, or asks "Что спросить у нейросети?";
3. within a minute after an answer that ended in a question, a phrase said
   without the wake word is a follow-up, unless it is a playback command or one
   of Alice's own phrases (time, date, weather, timers, alarms, reminders);
4. a rule: ask the model;
5. otherwise Alice answers. Unless the phrase is about music (which belongs to
   ytm-alice), Alice is muted until her answer appears in a state 3 frame: a
   proper answer is unmuted before she starts speaking, a helpless one is
   cancelled and goes to the model. Nothing of it is heard.

### Speaking the answer

There are two scenarios that make Alice speak ([06](06-quasar-ipc.md#two-ways-to-make-alice-speak)):
the dialog one opens the microphone afterwards but Yandex filters some topics;
the smart-home phrase one is never filtered but leaves the microphone closed.

- An answer that does **not** end in a question is read with the phrase
  scenario.
- An answer that **does** is sent through the dialog scenario, so that the user
  can reply. Alice stays muted until her frame shows our text. If it shows a
  refusal instead ("Да вы оригинал!"), the refusal is cancelled unheard, the
  answer is read with the phrase scenario, and then a muted "Слушаю." through
  the dialog scenario opens the microphone after all.

The follow-up window starts when listening actually opens, and every time window
is measured on a monotonic clock: the speaker's clock can jump by hours when it
synchronises.

### The model

Any OpenAI-compatible endpoint. Defaults: Gemini through Google's endpoint,
`models/gemini-flash-lite-latest`, falling back to `models/gemini-flash-latest`.
On HTTP 429/500/502/503/504 or a network error it retries twice with a pause,
the last time on the fallback model. The last four exchanges are kept for five
minutes. The journal names the model that answered.

The default instructions tell the model that it is Gemini, that Alice only
recognises the question and reads the answer, to answer in Russian in three or
four sentences of plain text, and to end with a short counter-question only when
the topic invites one.

## Installation

```
mkdir -p /data/llm
printf '%s' 'YOUR_API_KEY' > /data/llm/key && chmod 600 /data/llm/key
# copy the binary to /data/llm/llm-alice, chmod 755
/data/llm/llm-alice ask        # checks key and network, no voice involved
```

Then install [device/jbl-autostart/hooks.d/40-llm.sh](../../device/jbl-autostart/hooks.d/40-llm.sh)
([03](03-persistence-and-autostart.md)). It writes `llm-alice.service` with
`Restart=on-failure` into `/run` and restores mode 600 on the key before every
start, because `droproot_setup` resets it.

## Configuration

`/data/llm/llm-alice.json` is optional; every field overrides the default. Create
it on a computer and copy it over: Cyrillic pasted into PuTTY breaks.

```json
{
  "base_url": "https://generativelanguage.googleapis.com/v1beta/openai",
  "model": "models/gemini-flash-lite-latest",
  "fallback_model": "models/gemini-flash-latest",
  "retries": 2,
  "timeout_seconds": 20,
  "max_tokens": 400,
  "history_turns": 4,
  "history_minutes": 5,
  "system_prompt": "…",
  "triggers": ["спроси нейросеть", "спроси у нейросети", "спроси гемини"],
  "rules": ["объясни", "сравни", "почему", "как работает"],
  "helpless_answers": ["ничего не нашлось", "так просто не объяснить"],
  "web_answers_to_llm": true,
  "web_answer_markers": ["нашла ответ на", "дает такой ответ"],
  "alice_phrases": ["который час", "какая погода", "поставь таймер"],
  "reset_phrases": ["начни новый разговор", "новая тема"],
  "extra_params": {}
}
```

`extra_params` are added to every request; a `null` value removes a field. For
models that reject a parameter (as Gemini rejects `chat_template_kwargs`) this
is the switch.

## Journal and privacy

`/tmp/llm-alice.log` records every routed phrase, every answer, which model gave
it, which speech scenario read it, and Alice's own answers. It lives in RAM, is
capped at 256 KB and is gone after a reboot, but it does contain your questions.

## Limits

- The front LED ring cannot be kept "thinking" while the model works: it shows
  Alice's state, and her dialog is cancelled. See [13](13-hardware-research.md#leds).
- The model itself may refuse a topic; that is the model, not the speaker.
- A typical answer takes one to three seconds; an overloaded model, more.
- With the microphone already open, "Алиса" is sometimes not recognised at all.
  That is why `alice_phrases` stay with Alice even in a conversation.

## What was tried and dropped

- **Opening the microphone with an empty dialog request**: the session lights up
  but recognises nothing and hangs.
- **The cloud scenario of the smart-home API**, the way Home Assistant does it
  for speakers it cannot reach locally: the speaker's own account token is
  accepted, but the API wants a CSRF token on every change, creates a scenario in
  the account, and limits the text to 100 characters. The local phrase scenario
  above does the same without any of that.
- **`is_silent`, `reset_session`, `has_interaction`** on the server action: no
  effect.

## Compared with Quasar Bridge

| | Quasar Bridge | llm-alice |
|---|---|---|
| How it attaches | replaces `uniProxyUrl` and sits between the speaker and Yandex | reads local events next to Alice, no firmware settings changed |
| quasar 3.21 | does not work (protobuf only) | works |
| If it stops | voice stops working | Alice works as before |
| Who answers | rules and a classifier | triggers, rules, and fallback on Alice's own answer |
| Follow-ups | yes | yes, without the wake word |
| Extras | Home Assistant over MCP, AirPlay, web pages, sounds, installer | none |
| Listens on the network | yes, TLS | no |

On newer firmware Quasar Bridge is the richer choice. On 3.21 this is what works,
and it can be switched off without leaving a trace.
