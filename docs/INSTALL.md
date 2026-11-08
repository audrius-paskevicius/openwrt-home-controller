# Installation and operation

Read [the disclaimer](../DISCLAIMER.txt). This software controls physical equipment.
The first release targets OpenWrt 25.12 ARMv7 mvebu/cortexa9. Do not install its native
driver/helper binaries on another architecture.

1. Download the public signing key and both release APKs. Verify the signing-key
   fingerprint against the release notes through a trusted channel, then put the
   key in `/etc/apk/keys/home-controller-public.pem`.
2. Install the patched Telldus APK and the Home Controller APK with `apk add`.
   Install `sunwait` through OpenWrt's package manager for solar schedules.
3. Open **Services → Home Controller → Household access** in LuCI. Set a separate household username and
   password. Any non-empty password up to 256 characters is accepted. Click **Set
   household password**; its result appears beside the button. No default household
   password is shipped.
4. Connect a TellStick and add devices and optional groups in **Devices & groups**.
   Internal IDs are generated automatically. TellStick selection is automatic.
   LAN and detected VPN tunnels are selected by default; overrides are in **Advanced**.
5. Under **Schedules → Location and time zone**, set coordinates for solar events.
   Keep **Follow router automatically** to follow OpenWrt's
   time-zone setting. Ensure the router's date/time and NTP setup are correct.
6. Under **Advanced**, enable the service and apply. **Overview** shows status and a
   household-page link. Open `https://ROUTER-LAN-ADDRESS:8443/` (or your chosen
   port). Trust your router's certificate authority on household devices.

Home Controller creates its own HTTPS key. If the router has the `router-ca`
Easy-RSA helper, it requests a separate certificate from that local CA; otherwise
it creates a self-signed certificate that household devices must explicitly trust.
It never copies LuCI's private key. Use **Advanced → HTTPS certificate → Renew certificate**
before the one-year certificate expires or after changing listener addresses.
The certificate covers the router hostname and selected active addresses. Through
a VPN, use a covered address or hostname. A custom certificate and key can also be
installed by the administrator at `/etc/home-controller/tls.crt` and `tls.key`.

The package owns `/etc/tellstick.conf` once enabled. Do not combine it with another
application managing that file or the same USB transmitter. It does not modify any
other computer's FHEM installation. Device IDs and household radio addresses are
private configuration, never public examples or release defaults.

## Daily use

The household page shows individual devices first, then Scheduler, then groups.
Up/Stop/Down buttons are stacked vertically with a small status line for each control.
Phones use two columns. Wider screens add columns according to the available width.
Choose a **Symbol** in each device/group editor: Blind, Clock, Lamp, Plug, Fan or None.
Set the Scheduler symbol under **Household access → Household display**. Existing
devices/groups default to Blind, and Scheduler defaults to Clock. Symbols only
change appearance; choosing Lamp does not change a blind's available commands.
The small Sign out link is at the bottom. A remembered login
lasts for the administrator-selected number of days. Sign out ends that session;
LuCI can revoke all sessions or change the household account. Management is available
only through LuCI. Manual commands do not pause or disable later schedules.

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
Radio history and displayed last actions reset after a service restart; household
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
