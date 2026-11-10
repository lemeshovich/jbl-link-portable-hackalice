# Quasar IPC

`aliced` (inside `maind --yiod`) listens on `127.0.0.1:9875`. Every client that
connects receives Alice's state changes, and can send commands. This is how
ytm-alice hears voice commands and how it controls Alice.

## Framing

Each message is a 4-byte little-endian length followed by a protobuf
`QuasarMessage`. The same framing is used in both directions.

**Keep the connection open for a moment after writing.** If the client closes the
socket right after the write, `aliced` sometimes drops the message, apparently
because the connection is gone before it has registered the client. Waiting about
half a second before closing made every request arrive. Many of our "random"
failures came from this.

## Events from aliced

Field numbers below are for quasar 3.21.

`alice_state` (3):

| Field | Meaning |
|---|---|
| 1 | state: **0** idle, **1** listening, **2** speaking, **3** busy (thinking) |
| 2 | recognised text |
| 3 → 1 | text Alice is going to say or is saying (`vins_response.output_speech`) |
| 7 | request id |

`speech_kit_event` (43):

| Field | Meaning |
|---|---|
| 1 → 1 | wake word spotted, with the phrase, e.g. `алиса` |
| 2 | spotter begin (empty message) |

On this firmware the recognised text arrives in a state 1 frame and the final
state 3 arrives in a separate frame **without** text. Newer firmware sends both
together, which is what ytm-alice expected.

The cloud answer is visible in a state 3 frame (field 3 → 1) **before** Alice
starts speaking it. That is what makes it possible to silence a refusal without a
single syllable leaking.

## Commands that work on quasar 3.21

| Field | Message | Effect |
|---|---|---|
| 170 | `cleanup_directive_controller` (empty) | The firmware's own dialog cancel: in `yiod.log` it shows as `cancelDialog`, `stopDialog`, `clearRequestQueue`, `onPlayTtsCancel`. Stops speech and closes listening. Also cancels a request in flight. |
| 49 | `vins_server_action` → `server_action` (1) = JSON | Sends a server action as if it came from the dialog. With the `repeat_after_me` form update below, Alice says the given text in her own voice. |
| 79 | `vins_call` → `text_input` (1) | A text query, as if typed. Present in the schema; not needed in the end. |

The text-to-speech server action, the same one the
[YandexStation](https://github.com/AlexxIT/YandexStation) Home Assistant
integration sends over glagol:

```json
{"type":"server_action","name":"update_form","payload":{
  "form_update":{"name":"personal_assistant.scenarios.repeat_after_me",
    "slots":[{"type":"string","name":"request","value":"Включаю"}]},
  "resubmit":true}}
```

Notes from testing:

- Alice adds a varying lead-in to the displayed text ("Ваша фраза: «…»", "Цитируя
  вас, «…»"), but speaks only the requested text.
- With an empty value she says nothing, yet the request still replaces whatever
  the dialog was busy with. That releases a dialog stuck waiting for a reply that
  was dropped.
- After any such request the firmware opens the microphone for a few seconds by
  itself. Closing it needs a `cleanup_directive_controller` once listening has
  started. The `VinsServerAction` flags `is_silent` (5), `reset_session` (2) and
  `has_interaction` (3) changed neither the speech nor the listening.

## Commands that do not work on quasar 3.21

- Anything wrapped in `AliceCapability` (field 252/253), which newer ytm-alice
  uses for speech and dialog cancel. The string does not exist in this `maind`.
- `restart_vins_dialog` (86) with `need_cancel`, `sound_cancel` (40),
  `control_request.action_request` (101), and sending `speech_kit_event` (43)
  back to `aliced`: no observable effect.

## Getting the schema out of maind

Public `.proto` files exist ([quasar-protos](https://github.com/v3rt3p/quasar-protos)),
but their field numbers belong to newer releases. The exact schema of a given
firmware is compiled into `maind` as serialized `FileDescriptorProto` blobs.

1. Find them. `strings -o` in BusyBox prints offsets in **octal**:

   ```
   strings -o /system/vendor/quasar/maind | grep -E "quasar_proto\.proto|model_objects\.proto"
   ```

   Convert to decimal before using `dd` (`11576532` octal is 2,555,226).
2. Copy a region of about 1.5 MB around them with `dd` and bring it to a PC.
3. A descriptor starts with `0x0A`, a length byte and the file name, for example
   `\x0a\x12quasar_proto.proto`. From there, read top-level protobuf fields while
   they are valid `FileDescriptorProto` fields (numbers 1 to 12, wire types 0 or
   2); where that stops is the end of the blob.
4. Parse it with `google.protobuf.descriptor_pb2.FileDescriptorProto`.

On 3.21 this gives `quasar_proto.proto` with 37 messages and
`model_objects.proto` with 129. The descriptors themselves are not published here
because they come from a proprietary binary; the field numbers above are what
matters.

## Tools

In [tools/](../tools/):

- `alicedump`: prints every frame `aliced` sends, decoded field by field
- `alicesay [text]`: makes Alice say a text (default "Включаю"); `-c` uses
  `vins_call` instead; `-s`, `-r`, `-i` set the `VinsServerAction` flags
- `qcancel -f <field> [-E | -s <sub> -e | -v <n>]`: sends an arbitrary small
  `QuasarMessage`, used to find which fields have an effect
- `qplay`: sends a `MediaRequest` to the quasar audio client on port 9906
