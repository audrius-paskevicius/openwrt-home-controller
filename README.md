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
- Unprivileged household services, independent HTTPS key, and no cloud dependency.
- Patched Telldus driver preserving Hasta selflearningv2 unit-zero support.

The initial binary release targets **OpenWrt 25.12, ARMv7 mvebu/cortexa9**. Other
architectures require rebuilding. MQTT is a future extension, not a v0.1 dependency.

Read the [disclaimer](DISCLAIMER.txt), then the [installation guide](docs/INSTALL.md).
See [architecture](docs/ARCHITECTURE.md), [build instructions](docs/BUILD.md), and
[releases](https://github.com/audrius-paskevicius/openwrt-home-controller/releases).

No household credentials, coordinates, radio addresses, or private keys belong in this repository.
Local deployment information lives in the ignored `.private/` directory.
