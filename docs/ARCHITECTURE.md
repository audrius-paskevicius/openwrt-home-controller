# Architecture and boundaries

The control site and LuCI are different applications. A dedicated uHTTPd instance
serves Home Controller over HTTPS. It runs as `homecontroller`, with its own document
root and no ubus or LuCI endpoint. Its CGI gateway forwards only login, logout, state,
device-command and scheduler Start/Stop requests to a private Unix socket. The page response prohibits
framing and uses a restrictive Content Security Policy.

The Go controller uses the standard library; no Go modules are downloaded. The Go
runtime is compiled into the executable. Neither Perl nor Python is needed on the
router. A small C helper calls the Telldus library's Up, Down and Stop functions.
The patched Telldus daemon performs USB communication.

The LuCI page calls an administrator-only rpcd plugin. It validates configuration
before applying it. Control access never grants a router session or exposes
configuration, command execution, arbitrary file access, or radio addresses.
LuCI uses native `form.JSONMap`, tabbed sections, `GridSection` tables, modal editors,
input validation and the standard view action bar. Settings use the active LuCI
theme; only the disclaimer notice has scoped styling matching Wi-Fi Coordinator.
Control symbols are fixed SVG paths selected by a validated name,
never user-supplied SVG or external image URLs.

The JSON configuration has a separate root-only `pending.json` draft. RPC `stage`
validates and stores it; `save` activates validated settings in `config.json` and
removes the draft; `discard` removes only the draft. The running service reads only
the active file. RPC `get` distinguishes editable and active settings so pending
edits do not misrepresent service status. These methods require LuCI write access.
`control_order` contains typed device/group keys and a distinct Scheduler key;
unknown or duplicate keys are rejected, and omitted controls append in default order.

Both interfaces load package-installed copies of the canonical `DISCLAIMER.txt`.
They remember its exact text in local browser storage, with an in-memory fallback.
This acknowledgement is a UI gate, not an API authorization mechanism; authentication
and CSRF checks remain enforced independently.

## Authentication

The shared control account has an independently salted PBKDF2-HMAC-SHA256 password
hash (210,000 iterations). Random 256-bit bearer cookies are stored hashed on the
server. Cookies are Secure, HttpOnly, SameSite=Strict and host-only. Remembered sessions
expire after the configured duration (30 days by default); ordinary sessions have a
12-hour server lifetime and a browser-session cookie. Session records survive service
restarts. Password changes and administrator revocation invalidate existing sessions.
There is a 32-session bound and per-address/global login throttling.

State-changing requests require JSON, a same-origin request when Origin is present,
and an authenticated session's CSRF token. The token travels in Authorization because
uHTTPd's CGI implementation forwards only a fixed set of request headers. No CORS
permission is granted. Login requires JSON and is also origin checked.

## Network access

Listeners bind to selected logical network addresses and detected active VPN tunnel
addresses, never wildcard addresses. The default selected logical network is LAN.
WireGuard, OpenVPN/OpenConnect netifd interfaces and active TUN/TAP interfaces are
detected. Other routed VPNs can be selected by their logical network name in LuCI.
WAN/wan6 devices are explicitly excluded. IPv4 and non-link-local IPv6 are supported.
A separate nftables input chain drops traffic to the application port arriving on
other interfaces, before normal router input filtering. Existing firewall rules still
apply: a VPN zone that rejects router input needs an administrator rule allowing this
port. This package does not broadly open the VPN zone or change LuCI's listener.
Network hotplug events refresh the service bindings.

## Privilege and storage

The controller and control web server run as `homecontroller`. Their configuration
directory is root-owned; configuration and credential files are root-owned and group
readable. Only session/scheduler state storage and the runtime directory are writable by the service.
The separate control HTTPS private key is root-owned and group-readable by the
dedicated web server. LuCI's key and the router CA's key are never shared with it.
Privileged application of network rules, service configuration and account changes
is restricted to LuCI administration. The Telldus daemon is a separate upstream
service; it requires access to the USB controller and has its own privilege model.

The queue, last command states and last 100 results are in RAM. Sessions are written
only when their membership changes. The scheduler's stopped preference is written
when Start/Stop changes it. No continuous history is written to flash.
Unknown device position is explicit: a successful send is not a position measurement.

## Commands and schedules

One queue handles all control and scheduled commands. A newer command replaces a
queued command for the same device. Stop is placed before other queued movement.
An already executing radio call is bounded by a timeout; Stop cannot interrupt a
packet already being transmitted. Queued commands expire instead of replaying stale
movement. Group operations expand to individual device commands and are serialized.

The scheduler evaluates events every five seconds. It skips downtime/time jumps over
90 seconds, deduplicates events, and does not disable schedules after manual control.
Solar times are calculated by the optional `sunwait` executable with TZ=UTC and then
converted using the selected time zone. `router` follows OpenWrt's zone name. A solar
offset is applied before optional earliest/latest bounds. Missing solar events are
skipped unless a fallback time is set. Fixed-time schedules do not require sunwait.

Control Scheduler Stop persists before taking effect, cancels waiting scheduled
commands and prevents future scheduled sends. Manual commands remain available.
It cannot interrupt an in-flight radio call and does not send Stop to the blinds.
Start clips the evaluation interval to its resume time, avoiding missed-event replay.
The preference survives service restart/reboot; unreadable state leaves scheduling
stopped. Schedule definitions and individual enable flags remain administrator-only.
The next-event preview uses the same time-zone, solar and time-bound calculation as
execution. A runtime snapshot is refreshed every five seconds for LuCI status.

MQTT is not included. The controller queue is the intended integration
point for a future optional adapter, so that MQTT cannot bypass validation, ordering
or expiry. No broker or MQTT listener is installed by this package.
