# llm-alice

Answers from an LLM in Alice's own voice on quasar 3.21. How it works, routing,
configuration and limits: [docs/en/15](../docs/en/15-llm-alice.md).

```
sh build.sh
```

Runs the tests and builds `build/llm-alice` for ARMv7 with Go 1.27.3,
byte-identical to the released binary. Standard library only.

```
llm-alice serve          # the daemon
llm-alice ask [question] # one question, printed, no voice involved
llm-alice version
```
