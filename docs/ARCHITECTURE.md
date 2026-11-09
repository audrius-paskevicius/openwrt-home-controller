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

The Go controller uses the standard library plus pinned Pion DTLS and go-coap wire-codec modules for IKEA. The Go
runtime and these dependencies are compiled into the executable. Neither Perl nor Python is needed on the
router. A small C helper calls the Telldus library's Up, Down, Stop, TurnOn and TurnOff functions,
after checking the device's supported-method mask.
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
preference, IKEA gateway connection and diagnostics. Moving settings does not alter access enforcement.

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
readable. Only the state and runtime directories are writable by the service. The state
directory holds sessions, Scheduler preference, generated IKEA authentication and
remembered IKEA white/colour selections.
The pending configuration file is root-only and never loaded by the daemon.
The separate control HTTPS private key is root-owned and group-readable by the
dedicated web server. LuCI's key and the router CA's key are never shared with it.
Privileged application of network rules, service configuration and account changes
is restricted to LuCI administration. The Telldus daemon is a separate upstream
service; it requires access to the USB controller and has its own privilege model.

The queue, last command states and last 100 results are in RAM. Sessions are written
only when their membership changes. The scheduler's stopped preference is written
when Start/Stop changes it. No continuous history is written to flash.
TellStick receivers provide no physical position or power feedback: a successful
send is not a position measurement. IKEA state is reported separately by its gateway.

## Commands and schedules

Device `type` selects Blind, Switch, BH004 Dimmer, or IKEA behavior; a missing type means Blind for
older configurations. `protocol` is restricted to Hasta for blinds and Arctech
for switches/dimmers, with the corresponding fixed driver models `selflearningv2` and
`codeswitch`. Blinds retain numeric `house`; switches/dimmers use an A–P `house_code`
and zero numeric `house`. Unit ranges are validated separately. No arbitrary
protocol strings or radio-helper operations are accepted.

Local radio Groups derive their type from members and require distinct devices of
one radio type. IKEA groups are configured as IKEA targets instead.
Configuration validation checks schedule actions against targets; the command API,
queue and sender also reject incompatible actions. Public state exposes device/group
types, not radio addresses or protocol configuration. BH004 Toggle maps to the
driver's Off signal; Dim maps to its On signal, starting/stopping brightness
cycling while powered on. This mapping occurs only after type/action validation;
logs retain Toggle/Dim semantics. No state or brightness is inferred. Dimmers
and their groups support scheduled Toggle/Dim, with identical semantics to the
manual buttons. Absolute On/Off and brightness values remain unsupported. Scheduler is one built-in card and one scheduler engine, independent
of the number of schedule rules.

One queue handles all control and scheduled commands. A newer command replaces a
queued command for the same blind/switch. Dimmer presses are non-idempotent:
accepted commands retain FIFO order, with at most four queued presses per device.
The whole target is checked before enqueueing a group, so a full member queue
rejects the request without partial sends. Stop is placed before other queued movement.
An already executing radio call is bounded by a timeout; Stop cannot interrupt a
packet already being transmitted. Queued commands expire after 30 seconds instead of replaying stale
movement. Group operations expand to individual device commands and are serialized.

Control feedback is tracked per submitted target/request. Local radio Group result counters
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

## IKEA gateway

IKEA Home smart 1 remains responsible for pairing, discovery, names and group membership.
Home Controller imports no tiles automatically. An IKEA tile stores a gateway identity,
numeric resource ID, device/group kind and last-known capabilities, plus independent
local name, icon and order. Renaming a target in IKEA does not rename an existing tile.
Removed/unreachable targets remain configured and recover when available. Local radio
Group controls exclude IKEA; choose an IKEA group directly as an IKEA target.

The backend initiates authenticated DTLS-PSK connections to the configured private IP,
UDP 5684. No new listening ports, firewall exceptions, browser-to-gateway connections,
cloud login, shell interpolation or arbitrary CoAP paths are introduced. Only LuCI's
administrator RPC can set or reconnect the gateway. Control commands retain their
existing login, same-origin and CSRF requirements.

`/etc/home-controller/ikea/credentials.json` stores the address, learned MAC/hostname,
gateway identity, printed security code and any migrated client credentials. It is root-owned, mode 0640,
in a root:homecontroller 0750 directory: only root and the service account can read it.
Newly generated client credentials live in the service-owned, mode 0600
`/etc/home-controller/state/ikea-auth.json`. Neither file enters configuration JSON,
browser responses, diagnostics, source archives or APKs. They ARE included in the
router's protected OpenWrt backup of `/etc/home-controller/`; protect those backups.

The saved generated key handles reconnection; failed authentication can bootstrap a
new client key using the stored printed code. Reconnect clears the generated key and
restarts Home Controller. A changed printed code creates a different gateway identity,
so old tile IDs cannot silently select targets on a replacement gateway. Blank code
keeps the saved code. The code field shows eight placeholder stars when a code exists,
empty otherwise; only a boolean is returned to LuCI, never the saved code or its length.

On initial connection or upgrade, the administrator process records the gateway's MAC
and DHCP hostname from the router's lease file. At each discovery cycle the backend
looks for an unexpired lease matching that MAC, then falls back to the saved local
hostname and finally the previous private address. It refuses an address that the
lease table assigns to another MAC. Only private non-loopback addresses are accepted,
and DTLS authentication still verifies the endpoint. An IP change reconnects the
transport without changing tile identities or invalidating the saved client key.
No DHCP reservations, router DNS settings or firewall rules are changed. The standard
OpenWrt `/tmp/dhcp.leases` path is used. If that file is unavailable, hostname resolution
and the last address remain available. Address or hostname entry stays in Advanced;
the connection status shows the resolved IP.

Discovery polls devices and groups every 15 seconds, serialized with writes. After
a successful command it retains the previous complete snapshot for 750 ms from
the last member write, then schedules reads one second apart for eight seconds before
returning to normal polling. The browser reads the cached state every second for
ten seconds after submitting an IKEA command in that browser, then returns to its
five-second interval. Scheduled commands and other browsers do not trigger this
browser-local acceleration. Read
duration is additional to these intervals; inventories and commands share a network
lock, so this is not a one-second latency guarantee. This avoids
stacking long settling and polling delays; late gateway reports get prompt retries.
Remotes are omitted from controllable targets. Group capabilities are the
union of controllable members; On/Off affects plugs and lamps, while brightness,
white temperature and colour affect only supporting lamps. Writes are per member,
so a mixed group never sends light settings to a plug. Partial failures are reported;
there is no rollback or replay. Expired queued commands are not sent.

Group state is derived from member reports rather than the gateway's potentially stale
group attributes. Different member values show Mixed, and partial availability is shown
quietly. A new Mixed group result is confirmed on a later browser poll (at least four
seconds later); persistent differences remain visible. UI updates do not replace
the tile or change actively edited controls. Released sliders retain a pending
selection until feedback agrees, an error occurs, or eight seconds elapse; keyboard
focus alone does not indefinitely suppress updates.
Only committed slider selections send commands. IKEA tiles occupy one grid column,
with On/Off followed by horizontal B (brightness) and W (white temperature) sliders,
as supported. Colour-capable lights add a row of five dots: red, yellow, green, blue
and purple. Each sends a full-saturation colour command and preserves brightness.
Buttons, sliders and the colour row are 46px high with 8px gaps: four rows for B/W
lights, five for B/W/colour lights. Plugs remain two rows; Scheduler keeps its
three-row height. Colour dots and slider thumbs are both 28 px across; the dot row
uses a little of the side padding on screens up to 340 px to keep all five separate.
There are no rainbow slider tracks or external labels. Full names and values remain available
through hover descriptions and accessibility attributes. Plugs stay compact.

The status line shows Off, or On with brightness and a known white temperature, such as
`On 90% 2750K`. Colour mode omits Kelvin. The gateway does not expose a reliable mode
flag, so acknowledged white/colour commands record the mode in the private service-owned
`state/ikea-modes.json`. Changing hue/saturation/temperature externally invalidates that
knowledge when a new colour signature is observed; unknown mode omits Kelvin rather
than displaying a retained temperature as current. White-only bulbs always use white
mode. During a short post-command settling window, new reports establish the signature.
Mixed groups avoid a single misleading value.

The same private record retains separate white, hue and saturation selections. White commands
change the gateway's reported hue/saturation, and colour commands can change its
reported temperature. Those conversions do not move the white slider.
The authenticated API `controls` values describe the saved selections; measured values remain
separate and drive the status text. A hue or saturation command pairs its new
value with the other saved colour component, including rapid consecutive writes
before readback catches up. Zero saturation retains the selected hue for later
use. After the ten-second signature-settling window, a changed gateway colour
signature invalidates saved selections. Changes made by another controller during
that window cannot reliably be distinguished from this controller's own transition.

IKEA actions are On, Off, brightness (0–100%), white temperature (2200–4000 K), and
colour (#RRGGBB hue/saturation, brightness independent), hue (0–359 degrees), and
saturation (0–100%). Hue updates preserve saturation; saturation updates preserve hue.
This applies per member of a mixed group, including scheduled changes. Brightness >0, white and colour
turn lamps on; brightness 0 turns them off. These semantics are identical for manual
and scheduled actions. Pending changes coalesce only within the same setting;
brightness and colour scheduled together are both retained. The scheduler preview and
command history retain values. BH004 dimmers retain their separate Toggle/Dim behavior.

The constrained CoAP exchange uses library codecs with authenticated DTLS, confirmable
retransmissions, token/MID matching, separate-response acknowledgements and bounded
Block2 reads. It does not use the library's full UDP client, which exhibited an unaligned
64-bit atomic operation on the ARMv7 target. Network reads and response sizes are bounded.
