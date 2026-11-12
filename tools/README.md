# Tools

Small static helpers for the speaker. `sh build.sh` builds all of them.

| Tool | Use |
|---|---|
| `alicedump` | print every frame `aliced` sends on port 9875, decoded |
| `alicesay [text]` | make Alice say a text (default "Включаю"); `-f` uses the unfiltered phrase scenario; `""` releases a stuck dialog |
| `qcancel` | send a small `QuasarMessage` to `aliced`, for finding which fields work |
| `ipcclean dmix` | remove stale `dmix` shared memory segments (BusyBox has no `ipcrm`) |
| `qplay <url>` | ask the quasar audio client (port 9906) to play a URL |

Background: [docs/06](../docs/en/06-quasar-ipc.md).
