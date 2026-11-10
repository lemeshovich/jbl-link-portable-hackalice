# ytm-alice patch for quasar 3.21

`ytm-alice-quasar-3.21.patch` applies to
[ytm-alice](https://github.com/SPAWNRYS-ban/ytm-alice) at commit
`923e9acc2517184c447853e00e96eb600f5cef94` and changes `source/main.go`,
`source/ipc.go`, `source/cast.go`, adding `source/legacy_voice_test.go`.
What each change does and why: [docs/09](../docs/09-ytm-alice-port.md) and
[docs/10](../docs/10-voice-control.md).

```
sh build-ytm-alice.sh
```

Requirements and versions are listed in the script. It clones upstream, applies
the patch, and builds `build/out/ytm-alice` and `build/out/ytm-alice-resolver`,
byte-identical to the released binaries.
