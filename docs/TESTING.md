# Verification and test coverage

This report describes the current code and distinguishes automated checks from
previous hardware trials. Build and test commands are in [BUILD.md](BUILD.md).
Read the [disclaimer](../DISCLAIMER.txt); passing checks does not establish universal
hardware compatibility, physical position feedback or measured colour accuracy.

## 0.4.8 incomplete group reports

Browser regressions check that a partial IKEA group shows Status incomplete for
retained On, Off or mixed power values. A read-only gateway check after a reported
mains outage confirmed that reachability flags can remain stale; a successful GET
is not proof of contact with each lamp. The opt-in read test now also logs the
reported last-seen timestamp and age. It performs no lamp writes.

## 0.4.7 availability feedback

A browser regression checks that an unreachable IKEA group displays neutral
Unavailable despite older command/client errors, disables all controls, and still
shows genuine command failures for a reachable or partially reachable target.
The backend and radio components are unchanged.

## 0.4.6 documentation release

Version 0.4.6 reconciles the documentation with the implemented IKEA integration,
current tiles, feedback timing, radio-only groups and scheduling. It changes no
runtime behavior from 0.4.5. The application backend is unchanged from 0.4.4;
the radio helper and signed Telldus driver are unchanged from the published 0.3.2
release. The application APK is rebuilt with version 0.4.6-r1.

Release verification covers:

- Go tests and vet, JavaScript syntax, LuCI form-model tests and Chrome control-page
  and disclaimer tests. The GitHub workflow additionally runs Go's race detector
  on Linux and builds the ARM application and radio artifacts.
- Package signature verification and comparison of every extracted application
  payload file with its source or verified build input. The driver is verified
  separately using the same public signing key.
- An application source ZIP generated from the exact release commit with
  `git archive`, the unchanged driver corresponding-source archive, license notices,
  the canonical disclaimer and a checksum manifest covering all six other assets.
- Secret scans of public Git history and the staged public source. Build-time
  checks compare source and extracted package files against private gateway
  credentials without printing or exporting those credentials.
- Verification that the seven uploaded asset digests match the local release files.

Packaging this documentation release does not install it or restart the running
controller. The installed-interface and installation-preservation checks below
refer to 0.4.5 and earlier implementation revisions, not a new 0.4.6 installation.
No physical commands are needed for this documentation/package release.

## Automated coverage

| Area | Checks and main files |
| --- | --- |
| Authentication and access | Login validation, password hashing, session persistence/revocation, Origin/CSRF/content-type checks, selected network listeners and ingress restrictions; Go tests beside the implementation |
| Radio commands | Device/profile validation, address privacy, supported actions, request ordering, bounded dimmer presses, group outcomes and failure handling; `device_types_test.go`, `dimmer_test.go`, `control_status_test.go` |
| Scheduler | Weekly/solar calculations, time zones/DST, offsets/bounds/fallback, missed-event skipping, Start/Stop persistence and cancellation; `main_test.go`, `scheduler_control_test.go` |
| Configuration and TellStick selection | Draft/apply/discard, live USB preferences, unplug reset, handover and locking; `config_draft_test.go`, `controller_test.go`, `lock_linux_test.go` |
| IKEA protocol and discovery | Inventory/capability parsing, group membership, payload/range validation, scheduled values, independent queued settings, offline state, authenticated CoAP retry/ACK/blockwise reads, DHCP/MAC/hostname resolution; `ikea*_test.go` |
| IKEA feedback | Independent white/colour selections, rapid changes before readback, zero saturation, restart persistence, external signature invalidation, 750 ms initial pause and bounded one-second retry interval; `ikea_feedback_test.go` |
| LuCI forms | Type-dependent controls, naming/target selection, schedule values, Access fields, saved-code placeholder and credential exclusion; `tests/account-ui.cjs` |
| Browser controls | Phone/desktop geometry, On/Off and Toggle/Dim semantics, B/W sliders, five exact colour-preset requests, delayed readback, bounded Mixed filtering, temporary fast polling and idle recovery; `tests/browser-ui.cjs` |
| Disclaimer | First-use gate, acknowledgement persistence, changed text and unavailable storage/content; `tests/warning-ui.cjs` |

Browser/form fixtures use synthetic state and do not control real devices.
The control tests exercise widths of 320, 390 and 1280 px, with additional desktop
checks. Pointer and keyboard input, unavailable controls, readable dates and no
horizontal overflow are covered. The deployed 0.4.5 assets were also checked in
Chrome under their production CSP using intercepted state and blocked control writes:
all five colour dots matched the 28 px slider thumbs and fit without overlap.

The radio protocol test is separate from the Go suite. Its earlier target execution
passed 502 checks against compiled Telldus protocol code, including Hasta
selflearningv2 unit zero, other units, encoding/decoding and invalid frames. It
calculates packets without transmitting. It has not been rerun as a new RF trial
for a documentation-only release.

## Hardware and installed-interface evidence

Earlier TellStick trials on OpenWrt 25.12.6 ARMv7 checked Up/Stop/Down and scheduled
commands using an independent receiver, including a Hasta unit-zero device. That
verified received radio messages, not blind position. BH002 On/Off and BH004
Toggle/Dim behavior were tested separately; BH004's Off signal toggled power and
its On signal started/stopped dimming while lit. Another receiver did not respond,
so this is not a claim that every nominally compatible receiver works.

IKEA implementation revisions 0.4.0-0.4.3 were checked with an E1526 TRADFRI gateway.
Authenticated discovery returned controllable plugs, white bulbs, colour bulbs and
groups while excluding remotes. Authorized colour-bulb trials checked brightness,
white temperature, hue and saturation, including zero saturation and switching
between white and colour. Gateway reports confirmed the changes. The original
power, brightness, temperature, hue/saturation and XY readings were restored and
verified. These are reported values, not measurements of light output.

Native LuCI checks covered connected TellStick lists, preferences, IKEA connection,
placeholder stars with an empty actual code input, both Open controls links, target
selection and capability-dependent schedule fields. IP-change and reassignment
handling was tested with synthetic DHCP/DNS data; no real DHCP configuration was
changed to force an address change.

Installation checks through 0.4.5 compared existing control configuration, shared
login, Scheduler preference and network/firewall/LuCI settings before and after
upgrade. They remained unchanged. The generated IKEA key and saved gateway code
were preserved. Installed desktop/phone assets were checked under production CSP.
Read-only gateway checks confirmed connection and inventory; the 0.4.4 inventory
read completed in about a second in that environment. That is not a guaranteed
command-to-display latency: gateway reads, member writes and network time add to
the polling intervals documented in [ARCHITECTURE.md](ARCHITECTURE.md).

## Opt-in router diagnostics

Ordinary tests and CI skip these tests; credentials are read on the router itself.
Do not publish their private inventory, configuration or restoration files.

- `TestIKEARouterRead`: enable with `HC_IKEA_READ_TEST=1`. The optional
  `HC_IKEA_INVENTORY=/tmp/<file>` writes device/group inventory, not credentials.
  Inventory can still contain personal names and device identifiers.
- `TestIKEARouterControlAndRestore`: requires explicit physical-test authorization,
  `HC_IKEA_WRITE_TEST=1` and a numeric `HC_IKEA_WRITE_TARGET`. It changes the chosen
  bulb. A root-only restoration record is retained if restoration cannot be verified.
- `TestIKEARouterRestoreOnly`: with `HC_IKEA_RESTORE_TEST=1` and the same target ID,
  uses the retained record. Restoration tries original XY and H/S representations
  and checks the original readings before deleting the record.

## Limits

No automated fixture proves that a physical appliance moved, powered on or reached
a particular brightness. TellStick receivers provide no state feedback; IKEA reports
come from the gateway and may arrive late. Tests do not establish compatibility with
every bulb, gateway firmware, TellStick, router architecture or VPN arrangement.
Network/listener inspection is not an external penetration test. Home-screen asset
delivery has been checked; installation on every iOS/Android browser has not.
Earlier detailed release reports remain in Git history and their release snapshots.
