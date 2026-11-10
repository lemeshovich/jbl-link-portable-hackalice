# Overriding the firmware

## Bind mounts over a read-only root

The root filesystem cannot be remounted writable, but a file on `/data` can be
mounted over an existing file:

```
mount -o bind /data/path/replacement /system/vendor/quasar/quasar.cfg
```

Limits:

- only over a file or directory that already exists; a new path such as
  `/home/root/.asoundrc` cannot be created this way;
- the mount disappears at reboot, so it has to be redone from a hook;
- processes that already have the file open keep the old one until restarted.

## quasar.cfg

`/system/vendor/quasar/quasar.cfg` holds the daemon configuration, including
`uniProxyUrl` (where voice goes) and `voiceDialogSettings`. A copy kept in
`/data` and bind-mounted over the original takes effect after the quasar roles
restart.

The cloud also pushes settings at runtime and they win over the file: an edited
`uniProxyUrl` is overwritten in memory shortly after start. Pointing
`common.backendUrl` at an address that does not resolve (for example
`nosync.invalid`) stops that synchronisation, at the price of cloud features that
depend on it.

## user_config.dat

`/data/quasar/data/user_config.dat` is protobuf with the account configuration
that the cloud last sent. It can be edited in place if every changed string keeps
its byte length, so that no length prefix changes: for example a URL of the same
length padded with a dummy query such as `?pad=1234567`. Keep the original, and
protect the edited file with `chattr +i` or the next sync replaces it.

After a long time offline the token in it expires; re-pairing the speaker in the
Yandex app writes a fresh file.

These overrides were needed for [Quasar Bridge](11-quasar-bridge.md) and are not
used by the final setup.
