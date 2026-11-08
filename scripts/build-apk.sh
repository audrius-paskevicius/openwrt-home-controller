#!/bin/sh
set -eu
BASE=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$BASE"
: "${SIGN_KEY:?Signing key outside public source required}"
APK_TOOL=${APK_TOOL:-apk}
VERSION=$(cat VERSION)
ROOT=$(mktemp -d)
trap 'rm -rf "$ROOT"' EXIT
mkdir -p "$ROOT/usr/bin" "$ROOT/usr/libexec/rpcd" "$ROOT/usr/share/home-controller/www/api" "$ROOT/etc/init.d" "$ROOT/etc/home-controller" "$ROOT/etc/hotplug.d/iface" "$ROOT/etc/hotplug.d/usb" "$ROOT/www/luci-static/resources/view" "$ROOT/usr/share/luci/menu.d" "$ROOT/usr/share/rpcd/acl.d" "$ROOT/lib/upgrade/keep.d"
cp dist/home-controller "$ROOT/usr/bin/"
cp dist/home-controller-radio "$ROOT/usr/libexec/"
cp files/rpc "$ROOT/usr/libexec/rpcd/home-controller"
cp files/home-controller.init "$ROOT/etc/init.d/home-controller"
cp files/hotplug "$ROOT/etc/hotplug.d/iface/95-home-controller"
cp files/hotplug-usb "$ROOT/etc/hotplug.d/usb/95-home-controller"
cp files/config.json "$ROOT/etc/home-controller/config.json"
cp web/* "$ROOT/usr/share/home-controller/www/"
mv "$ROOT/usr/share/home-controller/www/index.html" "$ROOT/usr/share/home-controller/page.html"
printf '<!doctype html><meta http-equiv="refresh" content="0;url=/api/page"><title>Home Controller</title><a href="/api/page">Open Home Controller</a>\n' > "$ROOT/usr/share/home-controller/www/index.html"
cp DISCLAIMER.txt "$ROOT/usr/share/home-controller/www/disclaimer.txt"
for op in login logout state command page; do printf '#!/bin/sh\nexec /usr/bin/home-controller\n' > "$ROOT/usr/share/home-controller/www/api/$op"; done
cp luci/home-controller.js "$ROOT/www/luci-static/resources/view/"
cp files/menu.json "$ROOT/usr/share/luci/menu.d/luci-app-home-controller.json"
cp files/acl.json "$ROOT/usr/share/rpcd/acl.d/luci-app-home-controller.json"
printf '/etc/home-controller/\n/etc/tellstick.conf\n' > "$ROOT/lib/upgrade/keep.d/home-controller"
cp LICENSE DISCLAIMER.txt "$ROOT/usr/share/home-controller/"
find "$ROOT" -type d -exec chmod 755 {} \;
find "$ROOT" -type f -exec chmod 644 {} \;
chmod 755 "$ROOT/usr/bin/home-controller" "$ROOT/usr/libexec/home-controller-radio" "$ROOT/usr/libexec/rpcd/home-controller" "$ROOT/etc/init.d/home-controller" "$ROOT/etc/hotplug.d/iface/95-home-controller" "$ROOT/etc/hotplug.d/usb/95-home-controller"
chmod 755 "$ROOT/usr/share/home-controller/www/api/"*
mkdir -p dist
"$APK_TOOL" mkpkg --files "$ROOT" --sign-key "$SIGN_KEY" --info name:luci-app-home-controller --info version:"$VERSION" --info arch:armv7 --info license:GPL-3.0-or-later --info description:'Local household controls and scheduling with separate LuCI administration' --info 'depends:luci-base uhttpd telldus-core nftables jsonfilter' --script "post-install:$BASE/scripts/post-install" --script "post-upgrade:$BASE/scripts/post-install" --script "pre-deinstall:$BASE/scripts/pre-deinstall" --script "pre-upgrade:$BASE/scripts/pre-upgrade" --output "dist/luci-app-home-controller-$VERSION.apk"
