# Home Controller

A local OpenWrt household controller with a separate control page, LuCI administration,
weekly and solar schedules, and a replaceable Telldus transport. Initial device support
is Hasta selflearningv2 blinds (Up, Stop, Down). MQTT is reserved for a future optional adapter.

Household members get a small mobile-friendly control page with their own login and
remembered sessions. Administrators manage devices, groups, accounts and schedules in
LuCI. Access is restricted to selected LAN networks and configured VPN tunnels.

- Individual and group Up, Stop and Down controls; explicit unknown device position.
- Weekly schedules and optional `sunwait` sunrise, sunset, civil dawn and dusk events.
- Solar offsets, time bounds, and optional fallback when a solar event does not occur.
- Manual commands leave future schedules active; missed events are skipped.
- Household Scheduler Start/Stop, persistent stopped state, and the next scheduled action.
- Tabbed LuCI administration with device/group editors and guided schedule controls.
- Standard LuCI widgets and theme styling; configurable household symbols.
- Compact two-column phone controls and a desktop grid that uses the available width.
- Unprivileged household services, independent HTTPS key, and no cloud dependency.
- Patched Telldus driver preserving Hasta selflearningv2 unit-zero support.

The initial binary release targets **OpenWrt 25.12, ARMv7 mvebu/cortexa9**. Other
architectures require rebuilding. MQTT is a future extension, not a dependency.

Read the [disclaimer](DISCLAIMER.txt), then the [installation guide](docs/INSTALL.md).
See [architecture](docs/ARCHITECTURE.md), [build instructions](docs/BUILD.md), and
[releases](https://github.com/audrius-paskevicius/openwrt-home-controller/releases).
The [test report](docs/TESTING.md) distinguishes radio verification from physical
position feedback.

No household credentials, coordinates, radio addresses, or private keys belong in this repository.
Local deployment information lives in the ignored `.private/` directory.
