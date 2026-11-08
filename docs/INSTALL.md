# Installation and operation

Read [the disclaimer](../DISCLAIMER.txt). This software controls physical equipment.
Published binaries target OpenWrt 25.12 ARMv7 mvebu/cortexa9. Do not install its native
driver/helper binaries on another architecture.

1. Download the seven assets from the [latest release](https://github.com/audrius-paskevicius/openwrt-home-controller/releases/latest):
   both APKs, the application and driver source archives, `home-controller-public.pem`,
   `DISCLAIMER.txt`, and `SHA256SUMS`. Read the notice, then run `sha256sum -c SHA256SUMS`
   in that directory. It checks the other six files, not itself. Verify the public-key
   file's SHA256 against the release notes through a trusted channel, then put the
   key in `/etc/apk/keys/home-controller-public.pem`. Keep signature verification enabled.
2. Install the patched Telldus APK and the Home Controller APK with `apk add`.
   Install `sunwait` through OpenWrt's package manager for solar schedules.
3. Open **Services → Home Controller → Access** in LuCI. Set a separate shared username and
   password. Any non-empty password up to 256 characters is accepted. Click
   **Set password**; its result appears beside the button. Fresh packages ship with
   the service disabled, no devices or schedules, and no stored login credentials.
4. Connect a TellStick and add devices and optional groups in **Controls**.
   Internal IDs are generated automatically. TellStick selection is automatic.
   Under **Advanced → Allowed networks**, select the LAN and VPN networks that may
   access the control page. Fresh installations select LAN only. VPNs must be selected
   explicitly; leaving a VPN unchecked excludes it. WAN cannot be selected.
5. Under **Schedules**, set coordinates for solar events.
   Keep **Follow router** to follow OpenWrt's
   time-zone setting. Ensure the router's date/time and NTP setup are correct.
   Coordinates default to 0,0; they are not inferred from the router's time zone.
6. Under **General**, enable the service and click **Save & Apply**. This tab shows status and a
   control-page link. Open `https://ROUTER-LAN-ADDRESS:8443/` (or your chosen
   port). Include **https://** explicitly: this service has no HTTP listener or redirect.
   Trust your router's certificate authority on the devices you use to connect.

The short help text under **Advanced → HTTPS port** uses the same hover-help style
as Wi-Fi Coordinator. It explains that Home Controller's uHTTPd instance serves
the control page separately from LuCI's main web server. Hover, focus with the
keyboard, or tap to read the details; tap again or press Escape to dismiss them.

Editors change the form locally. **Save** stores an unapplied draft on the router;
it survives reloading the page and does not change the running service, even after
a restart. **Save & Apply** saves and activates the form. **Reset** discards both
local edits and the saved draft and reloads the active configuration. The native
LuCI action menu also offers **Apply unchecked**; both apply choices activate this
package's configuration directly, without UCI's network rollback mechanism.
Password changes, signing out devices and certificate renewal take effect immediately
through their own buttons. Their results appear beside those buttons.
Applying settings or changing the shared login restarts Home Controller. Remembered
sessions survive ordinary restarts; password changes and **Sign out all devices**
invalidate them. The scheduler's saved Running/Stopped preference is retained.

When upgrading from 0.2.3 or earlier, the saved network selections now determine
access immediately: VPN tunnels are no longer added automatically. Select and apply
any VPN you want to keep using before upgrading, or upgrade from LAN. At least one selected
network must have an active address for **Save**, **Save & Apply** and service startup,
even when preparing settings with the service disabled. VPN tunnel devices created outside OpenWrt's
network configuration must first be attached to a logical interface under
**Network → Interfaces**, then selected in **Allowed networks**.

The selector uses OpenWrt's logical network names. The standard `wan` and `wan6`
networks and their underlying devices are excluded; custom names/topologies are
not automatically identified as WAN. Existing router firewall rules still apply.
Interfaces sharing the same underlying device also share its incoming-traffic
permission. Local requests from the router itself remain permitted.

Both interfaces show the project disclaimer before first use. Acknowledgement is
remembered in that browser for the exact text. Changed wording asks again; blocked
browser storage means acknowledgement lasts only for the current page visit.
LuCI and the control page use separate browser origins, so each asks independently.
Another hostname, address, port or browser also needs its own acknowledgement.
Acknowledgement does not start or stop the service or Scheduler. Read the text again
from LuCI's **Experimental, AI-generated software. Use at your own risk.** tab
or the control page's **Disclaimer** link. The LuCI tab shows a read-only notice;
Save/Apply controls are hidden there. There is no separate Read warning button in General.
The package also installs the canonical notice at `/usr/share/home-controller/DISCLAIMER.txt`.

Home Controller creates its own HTTPS key. If the router has the optional `router-ca`
Easy-RSA helper, it requests a separate certificate from that local CA; otherwise
it creates a self-signed certificate that control devices must explicitly trust.
This package does not install or configure `router-ca`. If that helper is present
but issuance fails, renewal reports an error rather than switching to self-signed TLS.
It never copies LuCI's private key. Use **Advanced → HTTPS certificate → Renew certificate**
before the certificate expires or after changing the hostname or listener addresses.
The self-signed fallback lasts one year; certificates issued by `router-ca` use that
helper's configured validity. Renewal is manual and uses the active configuration,
so apply any address-related settings first.
The certificate covers the router hostname and selected active addresses. Through
a VPN, use a covered address or hostname. Network changes do not automatically
renew an existing certificate. A custom certificate and key can also be
installed by the administrator at `/etc/home-controller/tls.crt` and `tls.key`.

## Web server and access

Home Controller and LuCI use the same installed uHTTPd program in separate instances:

| Instance | Managed by | What it serves |
| --- | --- | --- |
| Main router web server | `/etc/config/uhttpd`; **Services → uHTTPd** when that LuCI module is installed | LuCI, including Home Controller's administration page |
| Home Controller web server | **Services → Home Controller**, its own init service | Control login, device buttons and Scheduler Start/Stop; HTTPS 8443 by default |

**Advanced → HTTPS port** and **Allowed networks** affect only the control page.
Its web server starts and stops with Home Controller. Removing the main uHTTPd
configuration affects LuCI access, not the separate control service. Both still
need the installed uHTTPd package. Changing or restarting either web instance does
not change the other instance's configuration.

The **Settings applied** message means the configuration was saved and a restart was requested.
It does not confirm that every restarted process succeeded. General's Enabled/Disabled
value reflects saved configuration; diagnostics show the last runtime snapshot.
If the control page is unavailable, check the selected networks, active addresses,
firewall, certificate and the system log. Runtime diagnostics may remain from an
earlier run when the service is stopped or startup fails. Keep a working management
connection when changing access settings: these settings have no automatic rollback.

## TellStick ownership

Enabling or restarting Home Controller rewrites `/etc/tellstick.conf` from its active
configuration and restarts `telldusd`. Preserve an existing configuration before
enabling it. Do not combine it with another
application managing that file or the same USB transmitter. It does not modify any
other computer's FHEM installation. Device IDs and control radio addresses are
private configuration, never public examples or release defaults.

## Daily use

Drag rows in LuCI's **Controls** table to arrange devices, groups and Scheduler
in any order, then click **Save & Apply**. Existing installations without an explicit
order retain devices first, then Scheduler, then groups. Newly added controls append
to the configured order.
Up/Stop/Down buttons are stacked vertically with a small status line for each control.
Phones use two columns. Wider screens add columns according to the available width.
Choose a **Symbol** in any row's editor, including Scheduler: Blind, Clock, Lamp,
Plug, Fan or None. Existing devices/groups default to Blind, and Scheduler defaults to Clock. Symbols only
change appearance; choosing Lamp does not change a blind's available commands.
The small Sign out link is at the bottom. A remembered login
lasts for 1–90 days as selected by the administrator (30 by default). Without
**Remember this device**, the server limits the session to 12 hours and the browser
receives a session cookie. Sign out ends that session;
LuCI can revoke all sessions or change the control account. Management is available
only through LuCI. Manual commands do not pause or disable later schedules.
Changing the remembered-session duration affects new logins; existing sessions keep
their original expiry unless revoked.

Times use a 24-hour clock regardless of browser language. Last-sent times on device
cards use `HH:mm` in the browser's local time zone. Dates and times elsewhere use
ISO 8601, for example `2026-11-08T22:10:00+01:00` (`Z` means UTC), with no AM/PM or
locale-dependent date order. Next scheduled actions use the configured schedule
time zone in both interfaces. LuCI's signed-in-device dates and numeric command
timestamps in diagnostics are displayed in the browser's local zone, including
the UTC offset. These display formats do not change schedule execution times.

Use your phone browser's **Add to Home Screen** command to save a shortcut. The
control page supplies a Home Controller icon, an Android web manifest and an
iPhone touch icon. Existing shortcuts may keep a cached image; remove and add the
shortcut again to pick up the new icon. This remains a local online control page,
not an offline controller.

The Scheduler card shows Running/Stopped and the next event in the configured router
or custom time zone. **Stop** cancels waiting scheduled commands and prevents future
scheduled sends, while manual controls remain available. It does **not** stop moving
blinds or interrupt a radio command already in flight; use the device/group Stop
buttons for that. **Start** resumes future events without replaying missed ones.
The stopped state survives service restarts and router reboots. With no saved
preference, Scheduler starts Running; new schedule definitions are disabled.
Individual schedule
enable flags stay unchanged; edit them in LuCI.

"Sent" means the Telldus API reported successful transmission. It does not prove
movement or the current position. Groups send one command per blind, sequentially.
Stop takes priority over commands still waiting, but cannot cancel a packet already
being transmitted. Pending commands expire after 30 seconds; a blocked transmitter
can therefore cause later commands in a large group to expire. The helper has an
eight-second alarm and the backend enforces a ten-second process timeout.

## Schedules

In the **Schedules** tab, add or edit a schedule using weekday selection, device/group
and action dropdowns, and either a fixed **HH:MM** time in 24-hour format or a solar
event. New schedules default to weekdays, 08:00 and Up, but remain disabled until
explicitly enabled. Solar
offsets are minutes before/after the event. Earliest/latest bounds clamp the computed
time. A missing polar solar event is skipped unless a fallback time is configured.
The fallback uses its own fixed time without applying the solar offset; the same
earliest/latest bounds still apply.
Technical calculation failures are skipped, not treated as polar fallback events.
Sunwait is an optional command-line calculation tool with no separate UI. The
Schedules tab reports whether it is installed; saving a solar rule does not install
it. Missing or failing calculations skip the affected events. After installing or
repairing sunwait, use **Save & Apply** to restart the service and clear cached
calculation failures. Fixed-time schedules do not need sunwait.

Schedules run without a browser or MQTT connection. Events missed during service
downtime are skipped. The scheduler rejects implausibly old startup clocks and skips
large clock corrections; it still relies on the router having the correct time.
Radio history and displayed last actions reset after a service restart; control
sessions remain valid until expiry or revocation.

## Hardware replacement

Connect a compatible original TellStick or TellStick Duo; no serial setting or save
is required. Existing blind addresses remain unchanged. When several TellSticks are
connected, LuCI identifies the one selected automatically. The current device is kept
while connected; otherwise the first available device in USB-port order is selected.
Old manually configured serial numbers are ignored and removed on the next save.
USB hotplug refreshes permissions and service access. New controller models or firmware
may require additional support and must be tested; this is not a promise of universal
Telldus hardware compatibility. No firmware is flashed by this package.

## Backup and removal

Keep a protected backup of `/etc/home-controller/` and `/etc/tellstick.conf`. They
contain credentials, sessions, TLS keys, coordinates and radio identifiers. OpenWrt's
backup keep-list includes these paths. Restore/install packages separately after a
firmware replacement; restoring configuration alone does not install executables.
The backup includes any unapplied draft. An upgrade preserves an existing private
configuration; the packaged generic defaults may appear as `config.json.apk-new`
and are not the active configuration.

Removing Home Controller stops its web/backend service and removes its dedicated
firewall table. It does not restore the previous `/etc/tellstick.conf` or stop/remove
the independently managed `telldusd` service. Review retained account files and
Telldus configuration before purging private files or reusing the transmitter with
another application.
