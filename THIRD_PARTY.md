# Third-party components

* Telldus Core: https://github.com/telldus/telldus, upstream LGPL licensing as supplied.
  The Hasta unit-zero patch is included in `patches/`; it retains default behavior
  for other units. OpenWrt's packaging patches are obtained from its packages feed.
* OpenWrt SDK and packaging: https://openwrt.org and https://github.com/openwrt/packages.
* sunwait: https://github.com/risacher/sunwait, GPL-3.0. Installed as a separate optional
  package. Its 0.9.1 CLI accepts years 0..99 despite conflicting help text.
* Go standard library/runtime: https://go.dev, BSD-style license. No external Go module
  dependencies. Runtime license accompanies release artifacts.
* LuCI and uHTTPd are existing OpenWrt packages, not copied into this repository.

Home Controller's application code is GPL-3.0-or-later. Generated builds remain
subject to the licenses of their constituent components.
