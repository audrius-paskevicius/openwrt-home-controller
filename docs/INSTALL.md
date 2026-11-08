# Installation and operation

Read [the disclaimer](../DISCLAIMER.txt). This software controls physical equipment.
The first release targets OpenWrt 25.12 ARMv7 mvebu/cortexa9. Do not install its native
driver/helper binaries on another architecture.

1. Download the public signing key and both release APKs. Verify the signing-key
   fingerprint against the release notes through a trusted channel, then put the
   key in `/etc/apk/keys/home-controller-public.pem`.
2. Install the patched Telldus APK and the Home Controller APK with `apk add`.
   Install `sunwait` through OpenWrt's package manager for solar schedules.
3. Open **Services → Home Controller → Access** in LuCI. Set a separate shared username and
   password. Any non-empty password up to 256 characters is accepted. Click **Set
   password**; its result appears beside the button. No default
   password is shipped.
4. Connect a TellStick and add devices and optional groups in **Controls**.
   Internal IDs are generated automatically. TellStick selection is automatic.
   LAN and detected VPN tunnels are selected by default; overrides are in **Advanced**.
5. Under **Schedules**, set coordinates for solar events.
   Keep **Follow router** to follow OpenWrt's
   time-zone setting. Ensure the router's date/time and NTP setup are correct.
6. Under **General**, enable the service and click **Save & Apply**. This tab shows status and a
   control-page link. Open `https://ROUTER-LAN-ADDRESS:8443/` (or your chosen
   port). Trust your router's certificate authority on the devices you use to connect.

Editors change the form locally. **Save** stores an unapplied draft on the router;
it survives reloading the page and does not change the running service, even after
a restart. **Save & Apply** saves and activates the form. **Reset** discards both
local edits and the saved draft and reloads the active configuration. The native
LuCI action menu also offers **Apply unchecked**; both apply choices activate this
package's configuration directly, without UCI's network rollback mechanism.
Password changes, signing out devices and certificate renewal take effect immediately
through their own buttons. Their results appear beside those buttons.

Both interfaces show the project disclaimer before first use. Acknowledgement is
remembered in that browser for the exact text. Changed wording asks again; blocked
browser storage means acknowledgement lasts only for the current page visit.
Acknowledgement does not start or stop the service or Scheduler. Read the text again
from LuCI's **Experimental, AI-generated software. Use at your own risk.** tab
or the control page's **Disclaimer** link.

Home Controller creates its own HTTPS key. If the router has the `router-ca`
Easy-RSA helper, it requests a separate certificate from that local CA; otherwise
it creates a self-signed certificate that control devices must explicitly trust.
It never copies LuCI's private key. Use **Advanced → HTTPS certificate → Renew certificate**
before the one-year certificate expires or after changing listener addresses.
The certificate covers the router hostname and selected active addresses. Through
a VPN, use a covered address or hostname. A custom certificate and key can also be
installed by the administrator at `/etc/home-controller/tls.crt` and `tls.key`.

The package owns `/etc/tellstick.conf` once enabled. Do not combine it with another
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
Plug, Fan or None. Existing
devices/groups default to Blind, and Scheduler defaults to Clock. Symbols only
change appearance; choosing Lamp does not change a blind's available commands.
The small Sign out link is at the bottom. A remembered login
lasts for the administrator-selected number of days. Sign out ends that session;
LuCI can revoke all sessions or change the control account. Management is available
only through LuCI. Manual commands do not pause or disable later schedules.

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
The stopped state survives service restarts and router reboots. Individual schedule
enable flags stay unchanged; edit them in LuCI.

"Sent" means the Telldus API reported successful transmission. It does not prove
movement or the current position. Groups send one command per blind, sequentially.
Stop takes priority over commands still waiting, but cannot cancel a packet already
being transmitted. A radio timeout can delay the next command up to its timeout.

## Schedules

In the **Schedules** tab, add or edit a schedule using weekday checkboxes, device/group
and action dropdowns, and a time picker or solar event. New schedules start disabled
until explicitly enabled. Solar
offsets are minutes before/after the event. Earliest/latest bounds clamp the computed
time. A missing polar solar event is skipped unless a fallback time is configured.
Technical calculation failures are skipped, not treated as polar fallback events.
The UI warns if sunwait is not installed. Reapply configuration after installing it.

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

Removing Home Controller stops its web/backend service and removes its dedicated
firewall table. Account settings and existing Telldus configuration should be reviewed
before purging private files or reusing the transmitter with another application.
