# Network

**English** · [Українська](../uk/12-network.md) · [Русский](../ru/12-network.md)

## YouTube does not stream to cloud addresses

If the speaker reaches the internet through a VPN or proxy whose exit is in a
data centre (here: a WireGuard tunnel to a cloud VM), YouTube Music search and
metadata keep working, but streams are refused. ytm-alice then reports a
misleading error:

> YouTube: для этого трека нужен вход и подтверждённый возраст аккаунта

The account and the login are fine. Route the speaker's own traffic directly
through your ISP (an exception in the router's policy routing) and the same
tracks play.

## DNS through the tunnel

While the speaker was routed through the tunnel, the router's DNS stopped
answering it, although `nslookup <name> 1.1.1.1` still worked. Things to know when
debugging this:

- `/etc/resolv.conf` is a symlink to `/run/resolv.conf`, written by `dhcpcd` when
  the lease is renewed; editing it does not survive a reconnect.
- Google names resolve to IPv6 addresses as well; if your network has no working
  IPv6, check which address a failing tool actually tried.
- Unstable latency to the router and failing DNS were symptoms of the tunnel
  route, not of the MTU, which was a false lead.
