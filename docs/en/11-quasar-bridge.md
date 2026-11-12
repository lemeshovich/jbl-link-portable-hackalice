# Quasar Bridge (and why it does not fit)

**English** · [Українська](../uk/11-quasar-bridge.md) · [Русский](../ru/11-quasar-bridge.md)

[Quasar Bridge](https://github.com/skrashevich/quasar-bridge) sits between the
speaker and Yandex's voice service (uniproxy): the speaker is pointed at the
bridge, which forwards to Yandex and can answer some requests with an LLM
instead. It was the first plan for this speaker. It does not work on quasar 3.21,
and the reason is worth knowing.

## What was done

- Built version 0.7.1 from source for ARMv7 and read through it before
  installing; the AirPlay PCM modules were included.
- Installed under `/data/opt/quasar-bridge`, listening on `127.0.0.1:8443` in
  `hybrid` mode. Its `control.sh` takes the speaker config from
  `QUASAR_DEVICE_CONFIG`, which helps on a read-only root: point it at a copy in
  `/data`.
- Pointed the speaker at it with the overrides from [05](05-overriding-the-firmware.md):
  `uniProxyUrl` in `quasar.cfg` and in `user_config.dat`, and `backendUrl` set to
  an unresolvable host so the cloud does not put the original URL back.
- LLM: Gemini through Google's OpenAI-compatible endpoint.

## Gemini answers 400

The bridge always adds `"chat_template_kwargs": {"enable_thinking": false}` to its
requests, a parameter meant for local servers such as vLLM or llama.cpp. Gemini
rejects it. Setting the key to `null` in the config removes it:

```json
"llm_chat_parameters": {"chat_template_kwargs": null}
```

## Why it cannot work here

The bridge speaks uniproxy in **protobuf**. Quasar 3.21 speaks it in **JSON**.

`voiceDialogSettings.useUniProxyProtobuf: true` looks like the switch, but on
this firmware it only changes part of the traffic: the audio stream goes out as
protobuf while the control channel stays JSON, and Yandex then rejects the
session, so recognition stops working altogether.

What replaced it on this firmware, and how the two compare:
[15, llm-alice](15-llm-alice.md#compared-with-quasar-bridge).

Reported upstream as
[issue #46](https://github.com/skrashevich/quasar-bridge/issues/46). Until the
bridge understands the JSON dialect, it is not an option for this firmware, and
it was removed from the speaker to free space on `/data`.
