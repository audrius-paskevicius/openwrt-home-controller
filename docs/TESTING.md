# Release verification

Version 0.2.2 uses native LuCI form maps, sortable grid sections, modal editors and
the standard view action bar. Synthetic RPC checks in the router's actual LuCI
runtime verify local editor changes, disabled new schedules, draft Save, Apply,
arbitrary device/group/Scheduler order, inline password errors, and Reset.
Location/time-zone fields are directly visible; configuration contains no
expandable disclosure sections. Backend tests verify that saved drafts cannot
change active settings, Reset discards them, Apply clears them, and invalid or
duplicate control-order keys are rejected.

Browser tests cover the unchanged phone controls and arbitrary mixed ordering.
Disclaimer tests cover canonical text, load failure/retry, exact-text storage,
changed text, reading again, and blocked local storage. Acknowledgement sends no
control commands. Shared-login handler tests check short passwords, failures,
duplicate submissions and feedback, and ensure credentials never enter saved
configuration. These synthetic tests do not transmit radio commands.

All 21 backend tests also passed on the target router. After the signed APK upgrade,
live LuCI checks confirmed the warning gate, six combined controls, visible schedule
settings, password feedback, a saved draft surviving page reload, and Reset removing
that draft. Configuration, credential and scheduler-preference files retained their
pre-upgrade SHA256 hashes; Scheduler remained stopped. The HTTPS control page was
checked under its production CSP for acknowledgement, login/logout, a 390-pixel
layout, its manifest and the new 180/192/512-pixel house icons. No radio commands
were sent. Existing phone shortcuts may need to be re-added to refresh cached icons;
physical phone shortcut installation is not covered by these browser tests.

Version 0.2.1 replaces custom LuCI styling with native tabs, widgets and table/form
markup. Browser checks cover the compact control layout, unchanged 46-pixel
buttons with 8-pixel gaps, configurable symbols (including None), and responsive
desktop columns. The symbol schema tests validate accepted choices, reject unknown
markup/URLs, and check configuration round trips and control API output.
All 19 backend tests passed on the target router. Live native LuCI checks covered
tabs, widgets, tables, schedule fields and inline password validation. Saving None
through LuCI removed a control symbol; the original setting was then restored.
Configuration and the user's scheduler preference matched their pre-test files
exactly. No blind commands were sent. At 390 pixels, the same four room names and
stopped scheduler measured about 75 pixels shorter than 0.2.0 without smaller buttons.
Shortcut assets include a manifest, scalable source, and 180/192/512-pixel PNGs;
browser delivery is checked, but installation on physical iOS/Android devices is
not part of the automated tests.

Version 0.2.0 redesigns both interfaces and adds control scheduler control.
Automated tests cover persistent Stop/Start, cancellation of waiting scheduled
commands while retaining manual commands, no missed-event replay, next weekly
event calculation, authentication/CSRF checks and failure to save scheduler state.
Browser tests use synthetic API/RPC responses to verify control ordering,
vertical touch controls and no horizontal overflow at 320/390/1024 pixels, scheduler
success/error feedback, LuCI tabs, generated device IDs, repeated editor saves,
schedule defaults and inline enable/error feedback. These tests do not transmit RF.

All 18 backend tests passed on the target router, including installed sunwait.
Live HTTPS checks verified scheduler authentication, CSRF/origin/method validation,
Stop surviving a service restart, the remembered session surviving that restart,
and Start resuming. The production control page loaded with its normal CSP and
stylesheet at mobile width; login, Start/Stop and Sign out worked. Actual LuCI tabs,
native editors and inline password validation were checked. A disabled schedule
was saved and deleted through LuCI; no test schedule remains. This release's live
checks issued no blind commands and did not modify the separate FHEM host.

Version 0.1.2 removes the minimum control-password length. Validation accepts
short non-empty passwords, rejects empty/overlong values, and counts Unicode
characters consistently. The LuCI handler test exercises inline saving, success,
validation and server/network error messages, plus duplicate-click prevention.
Tests use synthetic passwords; deployment preserves the existing control account.

Version 0.1.1 adds automatic TellStick selection. All 15 application tests passed
on the router, including replacement of a disconnected controller, stable selection
with several controllers, and migration of old serial settings. The installed
service selected the attached TellStick automatically. The actual LuCI view was
rendered with zero, one and multiple devices: no serial input is shown, and multiple
devices produce a read-only identification of the one in use. Multiple-device and
replacement cases were simulated; this update did not require radio transmissions.

Version 0.1.0 was exercised on an OpenWrt 25.12.6 mvebu/cortexa9 ARMv7 router with
an original USB TellStick. A separate TellStick Duo, attached to an unchanged FHEM
host, received the radio trials. No control identifiers or raw private captures
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
  old cookies. LuCI management and the control page are separate applications.
- Control page tested at desktop and 390-pixel mobile widths; all device controls
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
