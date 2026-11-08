# Installation and operation

Read [the disclaimer](../DISCLAIMER.txt). This software controls physical equipment.
The first release targets OpenWrt 25.12 ARMv7 mvebu/cortexa9. Do not install its native
driver/helper binaries on another architecture.

1. Download the public signing key and both release APKs. Verify the signing-key
   fingerprint against the release notes through a trusted channel, then put the
   key in `/etc/apk/keys/home-controller-public.pem`.
2. Install the patched Telldus APK and the Home Controller APK with `apk add`.
   Install `sunwait` through OpenWrt's package manager for solar schedules.
3. Open **Services → Home Controller** in LuCI. Set a separate household username and
   password (at least 12 characters). No default household password is shipped.
4. Select the TellStick serial shown under detected controllers, enter your devices
   and optional groups, and configure allowed LAN/VPN logical networks.
5. Set coordinates for your location. Keep time zone `router` to follow OpenWrt's
   time-zone setting. Ensure the router's date/time and NTP setup are correct.
6. Enable the service and save. Open `https://ROUTER-LAN-ADDRESS:8443/` (or your chosen
   port). Trust your router's certificate authority on household devices.

The installer copies the existing uHTTPd certificate and private key into the
service's protected directory. No new CA is installed in clients. The certificate
must cover the hostname/address used for the control page. Connect through a VPN
using the covered LAN hostname/address and a route to LAN. Router certificate renewal
requires using **Refresh HTTPS certificate from router** in Home Controller's LuCI page.

The package owns `/etc/tellstick.conf` once enabled. Do not combine it with another
application managing that file or the same USB transmitter. It does not modify any
other computer's FHEM installation. Device IDs and household radio addresses are
private configuration, never public examples or release defaults.

## Daily use

The household page has individual and group Up/Stop/Down buttons. A remembered login
lasts for the administrator-selected number of days. Sign out ends that session;
LuCI can revoke all sessions or change the household account. Management is available
only through LuCI. Manual commands do not pause or disable later schedules.

"Sent" means the Telldus API reported successful transmission. It does not prove
movement or the current position. Groups send one command per blind, sequentially.
Stop takes priority over commands still waiting, but cannot cancel a packet already
being transmitted. A radio timeout can delay the next command up to its timeout.

## Schedules

Choose weekdays, a device/group, an action, and a fixed time or solar event. Solar
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

Connect a compatible original TellStick or TellStick Duo, choose its detected serial
in LuCI, and save. Existing blind addresses remain independent of controller selection.
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
