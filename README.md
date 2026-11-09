# Home Controller

<!-- project-disclaimer:start -->
> [!WARNING]
> **EXPERIMENTAL, AI-GENERATED SOFTWARE — USE AT YOUR OWN RISK**
>
> This is a free, open-source project. The person behind it supplied the ideas, requirements, and direction; AI generated the code. That person cannot independently verify every aspect of its implementation or behaviour. Testing does not guarantee correctness, reliability, or safety.
>
> The software controls physical devices and runs scheduled actions. Errors may cause unexpected movement, missed or repeated actions, injury, property damage, network disruption, security weaknesses, data loss, or other harm.
>
> It is provided “as is,” without promised support, maintenance, or repair. To the fullest extent permitted by applicable law, the provider excludes warranties and liability arising from its use.
>
> Use only if you understand and accept the risks, have suitable backups and recovery arrangements, and can bear the resulting disruption, costs, or losses. Do not rely on it where failure could cause harm.
<!-- project-disclaimer:end -->

A local OpenWrt home controller with a separate control page, LuCI administration,
weekly and solar schedules, and a replaceable Telldus transport. Device support covers
Hasta selflearningv2 blinds (Up, Stop, Down) and Arctech code-switch receivers (On, Off).
MQTT is reserved for a future optional adapter.

Users get a small mobile-friendly control page with a shared login and
remembered sessions. Administrators manage devices, groups, the shared login and schedules in
LuCI. Access is restricted to explicitly selected LAN and VPN networks; WAN is excluded.

LuCI has **General**, **Controls**, **Schedules**, **Access**, **Advanced**, and the
full disclaimer tab. Device and schedule editors change a pending form; **Save**
stores a draft, **Save & Apply** activates it, and **Reset** discards unapplied changes.
**Access** starts with the control page's HTTPS port, allowed networks and certificate,
then its shared account and sessions. **Advanced** holds TellStick preference and diagnostics.
The shared login allows device commands and Scheduler Start/Stop; configuration
and account management stay in LuCI.

- Switch On/Off and Blind Up/Stop/Down controls, with groups of one device type.
  No physical position or power-state feedback is available.
- Dimmer is a disabled placeholder; one built-in Scheduler serves the system.
- Steady result messages; group feedback tracks its own commands, independently
  of later individual-device results.
- Weekly schedules and optional `sunwait` sunrise, sunset, civil dawn and dusk events.
- Solar offsets, time bounds, and optional fallback when a solar event does not occur.
- Manual commands leave future schedules active; missed events are skipped.
- Scheduler Start/Stop, persistent stopped state, and the next scheduled action.
- Consistent 24-hour times and readable `YYYY-MM-DD HH:mm` dates; exact timestamps
  remain in diagnostics and scheduler/session hover details.
- Native LuCI forms, tabs and sortable tables, with Save, Save & Apply and Reset.
- A live list of connected TellSticks, capabilities and the transmitter in use.
  Optional **Preferred TellStick** selection returns to Automatic when unplugged.
- Compact web-server help with Wi-Fi Coordinator-style hover, focus and tap support.
- One freely ordered list of devices, groups and Scheduler; configurable symbols.
- Compact two-column phone controls and a desktop grid that uses the available width.
- Home-screen shortcut icons for Android and iPhone.
- Unprivileged control services, independent HTTPS key, and no cloud dependency.
- Patched Telldus driver preserving Hasta selflearningv2 unit-zero support.

Published binaries target **OpenWrt 25.12, ARMv7 mvebu/cortexa9** and were tested on
OpenWrt 25.12.6. Other architectures require changing the build target and rebuilding.
MQTT is a future extension, not a dependency.

Read the [disclaimer](DISCLAIMER.txt), then the [installation guide](docs/INSTALL.md).
The control page uses its own uHTTPd instance (HTTPS port 8443 by default); its port
and allowed-network settings do not change LuCI's separate web server.
See [architecture](docs/ARCHITECTURE.md), [build instructions](docs/BUILD.md), and
[releases](https://github.com/audrius-paskevicius/openwrt-home-controller/releases).
The [test report](docs/TESTING.md) distinguishes radio verification from physical
position feedback.

The [latest release](https://github.com/audrius-paskevicius/openwrt-home-controller/releases/latest)
provides the signed application and driver APKs, their source archives, the public
signing key, the full `DISCLAIMER.txt`, and `SHA256SUMS`. The README and release
description display the full notice before installation instructions.

| Path | Contents |
| --- | --- |
| `cmd/home-controller/` | Backend, scheduler, authentication and administrator RPC |
| `luci/`, `web/` | LuCI administration and the separate control page |
| `src/`, `patches/` | Radio helper and Telldus Hasta unit-zero correction |
| `files/`, `scripts/` | Defaults, service hooks, builds and source checks |
| `tests/`, `.github/workflows/` | UI checks and CI; Go tests are beside the backend |
| `docs/`, `release/` | Documentation and public package verification key |

No personal credentials, coordinates, radio addresses, or private keys belong in this repository.
Local deployment information lives in the ignored `.private/` directory.
