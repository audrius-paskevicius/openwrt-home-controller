# Architecture and boundaries

The control site and LuCI are different applications. A dedicated uHTTPd instance
serves Home Controller over HTTPS. It runs as `homecontroller`, with its own document
root and no ubus or LuCI endpoint. Its CGI gateway forwards only login, logout, state,
device-command and scheduler Start/Stop requests to a private Unix socket. The page response prohibits
framing and uses a restrictive Content Security Policy.
The CGI gateway serves `/api/page` itself; static assets and the disclaimer are
served by that instance's document root. The control page shell is public, while
state, device commands and Scheduler operations require the separate control login.

Both web-server instances use the installed `/usr/sbin/uhttpd` executable, but have
separate process supervision, listeners and document roots. Home Controller's
init script starts its web instance alongside the Go backend. The normal OpenWrt
uHTTPd service reads `/etc/config/uhttpd` and serves LuCI independently; this package
does not create a section there. Removing its `main` section affects LuCI, while
removing the uHTTPd executable/dependency affects both applications.

The Go controller uses the standard library; no Go modules are downloaded. The Go
runtime is compiled into the executable. Neither Perl nor Python is needed on the
router. A small C helper calls the Telldus library's Up, Down and Stop functions.
The patched Telldus daemon performs USB communication.

The LuCI page calls an administrator-only rpcd plugin. It validates configuration
before applying it. Control access never grants a router session or exposes
administrator configuration, arbitrary shell execution, file access, or radio addresses.
The authenticated state API does expose the names, symbols and membership needed
for controls, last-command status, and the next scheduled action.
LuCI uses native `form.JSONMap`, tabbed sections, `GridSection` tables, modal editors,
input validation and the standard view action bar. Settings use the active LuCI
theme. The disclaimer notice and compact HTTPS-port help have scoped styling
matching Wi-Fi Coordinator, including light/dark tooltip colours. The short help
description exposes the full text on hover, keyboard focus or tap; tap again or
Escape dismisses it. The explanatory text is also exposed to assistive technology.
The Access tab starts with HTTPS port, allowed networks and certificate renewal,
followed by the shared account and sessions. Advanced contains transmitter
preference and diagnostics. Moving settings does not alter access enforcement.

Control symbols are fixed SVG paths selected by a validated name,
never user-supplied SVG or external image URLs.

Both interfaces explicitly format Gregorian dates with Latin digits and a 00–23
hour cycle. Device last-action labels show local `HH:mm`; scheduler and session
dates use readable ISO ordering, `YYYY-MM-DD HH:mm`. Scheduler previews retain the
configured schedule time zone; session dates use browser-local time. Scheduler and
session hover details retain the full timestamp and time zone. The compact scheduler tile keeps
the date and time together on their own line. Diagnostic command timestamps use
full UTC ISO 8601 strings, including seconds and `Z`. Raw API/runtime timestamp
fields retain their existing Unix-seconds or RFC3339 representations.

The JSON configuration has a separate root-only `pending.json` draft. RPC `stage`
validates and stores it; `save` activates validated settings in `config.json` and
removes the draft; `discard` removes only the draft. The running service reads only
the active file. RPC `get` distinguishes editable and active settings so pending
edits do not misrepresent service status. Draft/save/discard methods require LuCI
write access; `get` and session listing use the application's LuCI read permission.
Both `stage` and `save` require at least one selected network to have an active
address, even if the service is disabled. Applying only a changed transmitter preference
reconciles the radio without restarting the backend/web service. Other active
settings, including reapplying unchanged settings, request an asynchronous service
restart; a successful RPC response does not prove
that the restarted web/backend processes are healthy. No automatic configuration
rollback is implemented by these JSON configuration methods.

The optional `preferred_controller` is a currently connected USB serial; empty
means Automatic. The legacy `controller` field remains ignored. USB changes and
status reads remove missing preferences from active and pending settings without
discarding other edits. Save/Stage include the selected USB bus/device identity
as transient RPC metadata, preventing a stale page from restoring a preference
after unplug/replug. The configured preference is stored in the active/draft JSON;
selected-controller metadata is kept in the runtime directory. There is no historical
controller inventory. A reboot retains an explicit preference only
if that stick is present when configuration is reconciled.

Root configuration operations use a file lock to serialize Save/Apply, draft
changes and USB reconciliation. A separate radio file lock serializes transmission
with driver handover, and commands recheck expiry after waiting for it. Locks do
not follow symlinks on Linux. Handover regenerates `/etc/tellstick.conf`, refreshes
USB permissions and restarts only `telldusd`; unchanged selections do not restart
anything. Backend, scheduler and web sessions survive a radio-only handover.
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
The first control login must be set by an administrator; no credentials are shipped.

## Network access

Listeners bind only to addresses of explicitly selected, active logical networks,
never wildcard addresses. The default selected logical network is LAN. VPNs,
including WireGuard, OpenVPN and OpenConnect, must be selected in LuCI's **Allowed
networks** selector. Protocol and TUN/TAP/XFRM device names never grant access by
themselves. Tunnels outside netifd must be attached to a logical OpenWrt interface
before they can be selected. The same selections supply the ingress firewall's
allowed devices, so an unchecked VPN cannot reach the service through its LAN address.
Network aliases sharing one underlying device share its ingress permission.
The selector lists OpenWrt logical interfaces rather than independently classifying
their security role. Loopback remains permitted for requests from the router itself.
Devices assigned to the logical networks named `wan` and `wan6` are explicitly
excluded. This depends on those network names; it is not a classification of every
possible custom topology. IPv4 and non-link-local IPv6 are supported.
A separate nftables input chain drops traffic to the application port arriving on
other interfaces, before normal router input filtering. Existing firewall rules still
apply: a VPN zone that rejects router input needs an administrator rule allowing this
port. This package does not broadly open the VPN zone or change LuCI's listener.
Network hotplug events request a full service reload to refresh bindings, which
also resets in-memory command/group results. USB-only handovers use the lighter
radio reconciliation described above.
Changing the selected addresses does not renew an existing HTTPS certificate.
Certificate renewal uses the active configuration and the addresses available then;
apply network changes before manually renewing when certificate coverage changes.

## Privilege and storage

The controller and control web server run as `homecontroller`. Their configuration
directory is root-owned; configuration and credential files are root-owned and group
readable. Only session/scheduler state storage and the runtime directory are writable by the service.
The pending configuration file is root-only and never loaded by the daemon.
The separate control HTTPS private key is root-owned and group-readable by the
dedicated web server. LuCI's key and the router CA's key are never shared with it.
Privileged application of network rules, service configuration and account changes
is restricted to LuCI administration. The Telldus daemon is a separate upstream
service; it requires access to the USB controller and has its own privilege model.

The queue, last command states and last 100 results are in RAM. Sessions are written
only when their membership changes. The scheduler's stopped preference is written
when Start/Stop changes it. No continuous history is written to flash.
No physical position feedback is available: a successful send is not a position measurement.

## Commands and schedules

One queue handles all control and scheduled commands. A newer command replaces a
queued command for the same device. Stop is placed before other queued movement.
An already executing radio call is bounded by a timeout; Stop cannot interrupt a
packet already being transmitted. Queued commands expire after 30 seconds instead of replaying stale
movement. Group operations expand to individual device commands and are serialized.

Control feedback is tracked per submitted target/request. Group result counters
come from that group request, never an aggregation of arbitrary latest member
states. Replaced or scheduler-cancelled queued commands settle their original
request. An older in-flight completion cannot overwrite a newer device or group
request. Only the latest request per configured target is retained in memory;
`group_states` exposes sending outcomes without radio addresses. The browser keeps
existing result text through queue/transmit stages and updates completed results
only when they change. The status line reserves space even when blank.

The scheduler evaluates events every five seconds. It skips downtime/time jumps over
90 seconds, deduplicates events, and does not disable schedules after manual control.
Solar times are calculated by the optional `sunwait` executable with TZ=UTC and then
converted using the selected time zone. `router` follows OpenWrt's zone name. A solar
offset is applied before optional earliest/latest bounds. Missing solar events are
skipped unless a fallback time is set. Fixed-time schedules do not require sunwait.

The router zone name is resolved when the service is configured; OpenWrt system
configuration changes trigger a reload. Both successful and failed solar calculations
are cached by event/date until service restart or cache eviction. Fallback applies
only to a reported absent solar event, not to execution or parsing failures, and
does not receive the solar offset. Earliest/latest bounds apply afterward.

Scheduler Stop persists before taking effect, cancels waiting scheduled
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
