# Release verification

Version 0.1.1 adds automatic TellStick selection. All 15 application tests passed
on the router, including replacement of a disconnected controller, stable selection
with several controllers, and migration of old serial settings. The installed
service selected the attached TellStick automatically. The actual LuCI view was
rendered with zero, one and multiple devices: no serial input is shown, and multiple
devices produce a read-only identification of the one in use. Multiple-device and
replacement cases were simulated; this update did not require radio transmissions.

Version 0.1.0 was exercised on an OpenWrt 25.12.6 mvebu/cortexa9 ARMv7 router with
an original USB TellStick. A separate TellStick Duo, attached to an unchanged FHEM
host, received the radio trials. No household identifiers or raw private captures
are included here.

Passed checks:

- 13 Go tests on the router, with the installed sunwait executable. Coverage includes
  authorization, request validation, queue ordering, missed schedules, DST changes,
  solar bounds, polar fallback, cookies and login throttling. CI also runs Go's race
  detector, JavaScript parsing, source privacy checks and shell syntax checks.
- 502 checks against the compiled Telldus protocol code, without RF: synthetic
  selflearningv2 unit-zero packets, units 1 and 15, default unit behavior, checksums,
  pulse encoding, decoding, invalid frames and older selflearning behavior.
- Live Up/Stop/Down/Stop for four devices and group Stop: all 20 device commands
  succeeded and matching radio messages were received by the independent receiver.
  This includes a unit-zero device. Radio protocols may repeat packets.
- Live fixed-time Stop and bounded solar group Stop: all five device commands were
  received at their scheduled times. Test schedules are removed after verification.
- USB disconnect/reconnect through the kernel driver, automatic service recovery,
  and a successfully received command afterward.
- Actual HTTPS requests: unauthenticated access, GET operation attempts, missing
  CSRF token, foreign Origin, invalid action, management URL and wrong content type
  are rejected. An authenticated state request succeeds.
- Persistent sessions survive service restarts; administrative revocation rejects
  old cookies. LuCI management and the household page are separate applications.
- Household page tested at desktop and 390-pixel mobile widths; all device controls
  remain usable without horizontal overflow.
- Listener and firewall inspection: specific LAN/VPN IPv4 and IPv6 addresses only,
  with an input rule rejecting other interfaces. This is configuration inspection,
  not an external penetration test.
- Signed application and driver packages verify with the public release key and
  install on the target while preserving private configuration.

These tests do not prove physical blind movement or position: the installed
transmitter and blinds provide no position feedback. They also do not establish
compatibility with every TellStick model, router, VPN arrangement or future firmware.
See the [disclaimer](../DISCLAIMER.txt).
