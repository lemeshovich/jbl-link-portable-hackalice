# Root and SSH

**English** · [Українська](../uk/02-root-and-ssh.md) · [Русский](../ru/02-root-and-ssh.md)

This part follows the method published by lonsdaleite in
[PR #141](https://github.com/Marker284/yandex-station-extended/pull/141) to the
[hackalice](https://hackalice.me/) wiki. Read that guide first; this page only
adds what is specific to the 2021 firmware and what we did differently.

## Root through ADB

The firmware has a short USB ADB window during boot. With USB-C connected while
the speaker starts, a root shell is available for a few seconds:

```
adb wait-for-usb-device shell "touch /tmp/lock_activated_adb && ..."
```

`/tmp/lock_activated_adb` stops `maind` from closing USB ADB again until the next
reboot. `/data/network_adb` makes `adbd` listen on TCP 5555 as well, which is how
the first network access was obtained. The PR's `enable-adb.sh` creates these
markers and protects them with `chattr +i`.

Network ADB is an unauthenticated root shell for the whole LAN. Once SSH works,
remove it: take `chattr -i` off `/data/network_adb` and delete it. USB ADB at boot
remains as the way back in.

## SSH

Dropbear runs as a key-only server on port 2222, from `/data/jbl-ssh/`:

- built statically against musl with Zig, with password and PAM authentication
  compiled out: [device/ssh/build-dropbear.sh](../../device/ssh/build-dropbear.sh);
- started at boot by the PR's `ssh-service.sh` through the `10-ssh.sh` hook, which
  writes a transient systemd unit into `/run`;
- the whole `/data/jbl-ssh` tree is protected with `chattr -R +i`, so
  `droproot_setup` cannot loosen the permissions of the keys.

Use a dedicated ed25519 key pair and keep the host key backed up off the device.
The PR's `install-ssh.sh` performs these steps with checks; on this firmware the
same steps were done by hand.

## Moving files without scp

Dropbear here has no SFTP subsystem. Two ways that work from Windows `cmd`:

```
type file | ssh -p 2222 root@<speaker> "cat > /data/path/file"
```

and, for anything that must survive copy and paste, base64:

```
type file.b64 | ssh -p 2222 root@<speaker> "base64 -d > /data/path/file"
```

Always compare `sha256sum` on both ends before using a binary.
