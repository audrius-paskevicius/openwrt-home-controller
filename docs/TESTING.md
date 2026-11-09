# Release verification

These entries record checks performed for each release. Older entries describe
the interface and test suite at that time, not the current UI. Documentation is
checked against the current source separately; this does not repeat live radio trials.

## 0.3.2 receiver profile labels (2026-11-09)

- The Switch editor identifies BH002 / Arctech; the Dimmer editor identifies
  BH004 / Arctech. Each field is shown only for its matching device type.
- Form tests cover both labels and unchanged serialization to the Arctech
  code-switch protocol. Radio commands, schedules and control-page layout are
  unchanged from 0.3.1.

## 0.3.1 BH004 dimmer controls and compact tiles (2026-11-09)

- Physical BH004 trials confirmed that the Arctech code-switch Off signal toggles
  power in both directions, and On starts/stops brightness cycling while lit.
  The application exposes those signals as Toggle and Dim without claiming state
  feedback or absolute brightness. Another receiver did not respond; no broader
  receiver compatibility is claimed.
- Go tests cover semantic-to-radio mapping, supported actions, authenticated API
  access, address privacy, Toggle/Dim schedules, rejected absolute On/Off actions,
  preserved press order, bounded queues and atomic rejection of full group queues.
  Tests also pass on the router's ARM processor; Go vet passes.
- Form and native LuCI modal checks confirm Dimmer is selectable, no extra
  Scheduler can be added, and schedule actions follow the selected target.
  Serialization preserves letter addresses and dimmer group membership.
- Schedule weekday choices and summaries run Monday through Sunday; stored
  weekday numbers keep their existing meaning. Layout comparisons use normal
  single-line cards because wrapped error text can increase a card's height.
- Browser tests cover Toggle/Dim requests and groups, shorter two-button tiles,
  a Scheduler at least as tall as a three-button tile, and no horizontal overflow
  at phone and desktop widths. Existing status, account and disclaimer checks pass.
- The signed APK's 35 payload files match source/build inputs. The C helper and
  Telldus driver are unchanged. Installed assets were checked under production
  CSP using synthetic state with control writes blocked.
- Installation preserved configuration, credentials, Scheduler state, network,
  firewall and LuCI settings. A dimmer was then added through the existing
  administrator RPC, preserving all prior configuration fields. No schedules or
  extra physical radio commands were created by deployment.

## 0.3.0 Switch and Blind device types (2026-11-09)

- Added Arctech code-switch On/Off controls alongside Hasta blinds. Legacy devices
  load as Blinds. Tests cover both address ranges, wrong protocols/actions, mixed
  or duplicate group members, switch schedules and generated Telldus configuration.
- Browser tests verify switch/group buttons, schedule labels, stable feedback and
  the singleton Scheduler. Form tests cover serialization, type-specific unit
  validation, compatible group membership and changing schedule action choices.
- Native LuCI modal checks used synthetic, unsaved configuration: Dimmer is visibly
  disabled, a second Scheduler is unavailable, and schedule action choices change
  between On/Off and Up/Stop/Down when the target changes.
- The new C radio helper was built with the OpenWrt SDK in CI. Go tests pass locally
  and on ARM Linux, alongside Go vet, browser/disclaimer checks and source checks.
  Public API tests cover switch authorization, action validation and address privacy.
- The signed package's 35 payload files match source/build inputs. The driver is
  unchanged. A BH002 receiver's On command was physically confirmed during protocol
  identification; Off and BH004 dimming are not claimed as physically verified.
  No dimmer implementation is included.
- Installed 0.3.0 on the development router and added a Switch through the existing
  administrator RPC. All prior configuration fields were preserved, with only the
  new device and its control-order entry added. Credential, Scheduler-state and
  network/firewall/LuCI hashes were unchanged. Installed control assets passed
  desktop/phone checks under production CSP with synthetic state and writes blocked.

## 0.2.11 documentation and packaging audit (2026-11-09)

- Checked the current guides against LuCI form/model code, control rendering,
  command/group tracking, administrator RPC, USB reconciliation, network/TLS
  setup, defaults, service hooks, package builder and CI workflow.
- Corrected the README's date-format promise, the removed General status row,
  full-service versus radio-only Apply behavior, persisted TellStick preference,
  status refresh/reset semantics and build verification commands. Historical test
  entries retain the behavior and evidence of their own releases.
- This release changes documentation and package version only. Backend, radio
  helper, driver and interface assets are reused unchanged from 0.2.10. The signed
  APK is built and signature-verified; its extracted payload is checked against
  source and the previous verified package. Application source, documentation
  links, canonical disclaimer and all release checksums are checked separately.
- This packaging audit does not install the new APK or restart the controller.
  Runtime and physical-radio evidence remains attached to the releases actually
  tested; no new hardware trial is claimed.

## 0.2.10 quieter control feedback and Access layout (2026-11-09)

- HTTPS port, allowed networks and certificate renewal are first in Access.
  Form tests verify their location and preservation of port/network values when
  saving. The access policy, listeners and certificate implementation are unchanged.
- Group results track the actual group request. Tests cover unrelated individual
  successes/failures, partial group results, replaced commands, cancelled schedules
  and older completions arriving after newer commands.
- Chrome checks verify stable text through command submission, queue stages and
  repeated polls, independent group messages, partial-result feedback and layouts
  from 320 to 1920 pixels. No routing-to-device-results prompt remains.
- Go tests, Go vet, JavaScript syntax/form tests and disclaimer checks pass.
  ARM Linux tests run on the router with synthetic radio results; no physical
  blind commands are sent by this verification.
- The signed package was installed and all 35 payload files matched their inputs.
  Native LuCI Access/Advanced contents and phone/desktop layouts were checked.
  Installed control assets were checked under production CSP with synthetic state
  and control writes blocked. Configuration, credentials, Scheduler state and
  network/firewall/LuCI settings retained their hashes. Secret scans passed.

## 0.2.9 connected TellSticks and automatic fallback (2026-11-09)

- General lists connected models, serials, sending/receiving capabilities and the
  selected transmitter. The duplicate configuration-only Service row is removed.
  Advanced offers Automatic and currently connected sticks only.
- Go tests cover explicit preference, sticky automatic selection, fallback,
  disappearance from active and staged settings, preservation of unrelated edits,
  reconnect without restored preference, and rejection of stale USB identities.
  Linux tests also verify radio-lock serialization and symlink rejection.
- Tests pass on Windows and on the router's ARM Linux CPU. JavaScript/form tests,
  Go vet, browser layout/disclaimer tests and shell syntax checks pass.
- Native LuCI was checked in Chrome on the installed router at desktop and
  390-pixel widths. Both physical sticks appeared with the correct capabilities.
  Switching between them restarted only the Telldus driver; backend and web PIDs
  and Scheduler state remained unchanged. The original transmitter and Automatic
  were restored afterwards.
- Simulated USB removal/add events on the router cleared the preference without
  restoring it. Synthetic RPC responses in native LuCI verified option removal,
  automatic reset and option reappearance. These checks did not physically unplug
  hardware, and no blind commands were sent; fallback radio transmission is not
  claimed as physically verified.
- The signed APK's 35 payload files match their source/build inputs. Package
  signature, public-source checks and secret scans pass. Router network, firewall,
  LuCI web-server settings and shared-account credentials were preserved.

## 0.2.8 readable dates and disclaimer styling (2026-11-09)

- Scheduler previews and session dates use `YYYY-MM-DD HH:mm`. The control tile
  puts the timestamp on its own unbroken line without a visible time-zone suffix.
  Configured schedule zones and browser-local session times are preserved; exact
  timestamps remain in metadata/hover details and diagnostic JSON.
- Control-page checks passed at 320, 390, 1280 and 1920 pixels, across US/Arabic
  locales, midnight, both repeated-hour instants, UTC and fractional/negative
  offsets. LuCI form checks include readable session dates and precise diagnostics.
- Home Controller's first-use and permanent notices use the amber Wi-Fi Coordinator
  styling. Browser checks passed for exact-text persistence, changed-text prompts,
  blocked storage and read-again access. Canonical project wording is unchanged.
- All 35 signed APK payload files match source or unchanged backend/radio binaries.
  Source checks and package signature verification passed. The backend and radio
  algorithms are unchanged; physical device commands were not exercised.
- The signed package was installed on the development router. Configuration,
  credentials and Scheduler state hashes were preserved, and the service resumed.
  HTTP reads of the installed control page, JavaScript and stylesheets match the
  verified sources, including the versioned asset URLs.

## 0.2.7 time display (2026-11-08)

- Control-page tests verify `HH:mm` command times and ISO 8601 date-times with
  US and Arabic browser locales, midnight, the repeated daylight-saving hour,
  negative offsets, a 45-minute offset and UTC. Layout and warning tests passed.
- LuCI form tests verify the configured schedule time zone and browser-local
  session dates. The installed LuCI session table displayed ISO timestamps.
- The deployed control-page assets were checked under production CSP with synthetic
  state, US locale and a 390-pixel viewport. No control requests were sent.
- The signed 0.2.7-r1 package preserved configuration, credentials and Scheduler
  state during upgrade. Backend and radio binaries are unchanged; formatting is
  confined to the interfaces and does not alter schedule execution or API data.

## 0.2.6 documentation and package verification (2026-11-08)

- Audited the guides against the current UI, administrator RPC, network selection,
  TLS generation, service hooks and package builder. Clarified the two uHTTPd
  instances, help styling, save/restart semantics, network prerequisites, certificate
  renewal and package removal behavior.
- Built and signature-verified the 0.2.6-r1 application APK. Checked its extracted
  backend, radio helper, LuCI assets, control assets, service hooks and canonical
  disclaimer against the release inputs. Backend and radio artifacts are reused
  from 0.2.5; this release changes documentation and package version only.
- Verified local documentation links, matching disclaimer text, source-archive
  contents and the six-file release checksum manifest. The source ZIP contains a
  `home-controller/` top-level directory and corresponds to the release commit.
- Runtime/CI and physical-radio evidence remains in the entries for the versions
  actually tested. This package was built without installing it, restarting the
  controller or changing live settings; no new radio trial is claimed.

## 0.2.5 web-server help (2026-11-08)

- Added compact help under **HTTPS port**, matching Wi-Fi Coordinator's tooltip
  presentation. Verified on the installed router using desktop hover, keyboard
  focus/Escape, and a 375-pixel touch viewport with tap-to-open/tap-to-close.
- The tooltip remained within the narrow viewport and loaded the packaged help
  stylesheet. JavaScript syntax and existing form-handler checks passed.
- The signed package upgrade preserved configuration, credentials and Scheduler
  state. Backend and radio binaries are unchanged from 0.2.4; no device commands
  were sent and no settings were saved during help verification.

## 0.2.4 network selection fix (2026-11-08)

- All 22 Go tests passed on the target ARM router, including regression cases for
  selected and unselected VPNs, LAN exclusion, offline interfaces, IPv6-only
  addresses, WAN/WAN6 rejection and aliases of WAN devices.
- Native LuCI **Allowed networks** was exercised on the router: selecting a VPN
  added its HTTPS listener and ingress firewall permission; deselecting it removed
  both. The original LAN-only selection was restored and persisted across reload.
- JavaScript syntax and form-handler tests passed, including saving and removing
  a selected VPN, empty-selection validation and WAN option exclusion.
- The signed package upgraded from 0.2.3-r1 to 0.2.4-r1. Configuration, credentials
  and saved Scheduler state matched their pre-upgrade checksums. The Scheduler
  remained stopped during live network tests; no device commands were sent.

## 0.2.3

Version 0.2.3 adds a permanent disclaimer tab matching Wi-Fi Coordinator's warning
label, amber notice, and emphasized closing paragraph. Live native LuCI checks
verified the canonical text, removal of General's Read warning button, and the
Save bar hiding on the disclaimer and returning on settings tabs, including after
page reload. Existing acknowledgement tests and form-handler checks passed. The
signed package upgrade preserved configuration, credentials and Scheduler state.

## 0.2.2

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

## 0.2.1

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

## 0.2.0

Version 0.2.0 redesigns both interfaces and adds Scheduler Start/Stop to the control page.
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

## 0.1.2

Version 0.1.2 removes the minimum shared-password length. Validation accepts
short non-empty passwords, rejects empty/overlong values, and counts Unicode
characters consistently. The LuCI handler test exercises inline saving, success,
validation and server/network error messages, plus duplicate-click prevention.
Tests use synthetic passwords; deployment preserves the existing control account.

## 0.1.1

Version 0.1.1 adds automatic TellStick selection. All 15 application tests passed
on the router, including replacement of a disconnected controller, stable selection
with several controllers, and migration of old serial settings. The installed
service selected the attached TellStick automatically. The actual LuCI view was
rendered with zero, one and multiple devices: no serial input is shown, and multiple
devices produce a read-only identification of the one in use. Multiple-device and
replacement cases were simulated; this update did not require radio transmissions.

## 0.1.0

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

## IKEA integration (0.4.0)

Verified on an E1526 gateway and the ARMv7 router: authenticated device/group reads;
capability detection for plugs, white bulbs, colour bulbs and mixed groups; remotes
excluded from controllable targets. With explicit user authorization, a colour bulb
accepted brightness, hue/saturation and white-temperature commands. Gateway reports
confirmed each change. Its original power, brightness, hue, saturation and white
temperature were restored and verified. This confirms reported values, not measured
light output or colour accuracy. Other device models remain untested.

Automated checks cover action/range validation, scheduled values, independent queued
settings, mixed-group parsing, offline state, API authentication/CSRF, credential
exclusion, confirmable retry/separate responses and blockwise CoAP reads. Chrome checks
cover fixed four-row tiles at 320/390/1280px, commit-only sliders, stable status text,
unavailable controls and existing radio/scheduler/disclaimer behavior. Native LuCI was
checked with unsaved synthetic targets/rules for target selection, initial naming,
colour input and action-dependent schedule fields.

`TestIKEARouterRead` requires `HC_IKEA_READ_TEST=1` on the router and reads existing
credentials in place. Optional `HC_IKEA_INVENTORY=/tmp/<file>` writes only inventory.
`TestIKEARouterControlAndRestore` is never enabled in normal tests or CI: it requires
explicit physical-test authorization plus `HC_IKEA_WRITE_TEST=1` and a numeric
`HC_IKEA_WRITE_TARGET`. It changes the selected bulb and restores its original values;
a root-only temporary restoration record is retained if restoration cannot be verified.

The signed 0.4.0 APK was installed and all payload files matched the build inputs.
Existing control configuration, shared login, scheduler preference and router network,
firewall and LuCI configuration remained unchanged. Installed discovery and Reconnect
were verified; reauthentication generated a service-owned private key file using the
stored code. Installed desktop/phone assets were checked under production CSP with
synthetic state and control writes blocked. No IKEA tiles or schedules were added by
these checks. Source and extracted payload scans found no deployment credentials.
