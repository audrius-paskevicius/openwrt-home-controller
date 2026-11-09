# Build and publication

Read the [full disclaimer](../DISCLAIMER.txt) before building or installing.

The repository is self-contained: application and UI source, radio helper, driver
patch, package scripts, tests, documentation and public package verification key are
included. External inputs are the Go toolchain, modules pinned in go.mod/go.sum, OpenWrt SDK/package sources and
Telldus upstream source. The build scripts target OpenWrt 25.12 on ARMv7
(mvebu/cortexa9). Other targets require changing the SDK, Go architecture settings
and APK architecture metadata, then rebuilding the backend, driver and radio helper.

Run the commands from the repository root on a Linux x86-64 host with **Go 1.26 or
later**, **Node.js 24 or later**, Python 3, Bash, Git, make/GCC and the OpenWrt SDK's
host build prerequisites, tar and zstd. CI uses Ubuntu 24.04 and Go 1.27. Native
apk-tools v3 with `mkpkg` and `adbsign` is also needed for packaging; ordinary older
apk-tools builds may not provide those applets.

```
go test -race ./...
go vet ./...
python3 scripts/check-source.py
node tests/check-js.cjs
node tests/account-ui.cjs
sh -n scripts/build-apk.sh
sh -n files/home-controller.init
sh -n files/hotplug-usb
mkdir -p dist
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/home-controller ./cmd/home-controller
bash scripts/build-radio.sh
```

The radio build downloads the SHA256-pinned OpenWrt 25.12.6 mvebu/cortexa9 SDK,
fetches its package feeds, upgrades Telldus to the checksum-pinned 2.1.3-beta1 source,
and applies the Hasta selflearningv2 unit-zero correction. Feed revisions are not
pinned by this script; this is not a promise of byte-identical future rebuilds.
Outputs include `dist/home-controller-radio`, the Telldus APK and corresponding-source
archive, and `dist/hasta-test`. The source and patch must accompany distributed
driver binaries.

Create your own signing key outside the public tree:

```
mkdir -p ../home-controller-signing/public
chmod 700 ../home-controller-signing
go run scripts/genkey.go ../home-controller-signing/private.pem ../home-controller-signing/public/local-public.pem
SIGN_KEY="$(pwd)/../home-controller-signing/private.pem" APK_TOOL=/path/to/apk.static sh scripts/build-apk.sh
```

The application builder reads [VERSION](../VERSION), requires the backend and radio
helper in `dist/`, and creates `dist/luci-app-home-controller-VERSION.apk`. It copies
the canonical disclaimer into the package and both interfaces, and installs the
LuCI disclaimer and help stylesheets. The Markdown guides are distributed in the
repository and application source archive; they are not installed by the APK.
The builder does not generate
the public key, release notes, source ZIP or checksum manifest. Keep the repository's
published verification key unchanged when building with your own key.

Re-sign a copy of the SDK-built driver APK with the same key used for your application
APK. Signing tools may modify a file even on failure. The SDK uses a temporary key,
so allow that input only during re-signing:

```
cp dist/telldus-core-2.1.3_beta1-r10.apk dist/telldus-core-local.apk
/path/to/apk.static adbsign --allow-untrusted --reset-signatures \
  --sign-key ../home-controller-signing/private.pem dist/telldus-core-local.apk
/path/to/apk.static --keys-dir ../home-controller-signing/public verify \
  "dist/luci-app-home-controller-$(cat VERSION).apk" dist/telldus-core-local.apk
```

Verification and installation must use the trusted public key, without
`--allow-untrusted`. Publish only sanitized release files. The radio build also
produces the patched corresponding-source archive and the `hasta-test` executable;
run that test on the target (it calculates packets without transmitting).
Your corresponding public verification key belongs in `/etc/apk/keys/` on the target.
The repository's public key cannot verify packages signed with your private key.

`node tests/browser-ui.cjs` exercises the control page in a disposable headless Chrome
listening on localhost:9222 (Node 24 or later). It injects synthetic API responses
and never contacts a router or sends radio commands. It checks phone/desktop layouts,
date formatting, control ordering, scheduler interaction and stable, independent
group feedback. `node tests/account-ui.cjs` checks the LuCI form model, Access field
placement, password feedback, Save/Apply/Reset and connected TellStick preferences
with stubbed form components. `node tests/warning-ui.cjs` checks acknowledgement and storage behavior
for both interfaces. Native LuCI rendering is checked separately on the target.
Set `UI_SCREENSHOTS=1` to save control screenshots under ignored `.private/`.
Set `CHROME_PORT` when the disposable browser uses a port other than 9222.
The Chrome startup and teardown commands used in CI are in
[the build workflow](../.github/workflows/build.yml). CI tests and compiles the
backend and radio artifacts; it does not sign or publish release APKs.

## Release visibility and contents

Place the complete canonical [DISCLAIMER.txt](../DISCLAIMER.txt) at the top of the
README and release description in a GitHub `> [!WARNING]` block, before feature or
installation text. Keep the wording identical to the packaged notice. The README
block is delimited by `project-disclaimer` comments. Publish the notice as a separate
download too, as in the Wi-Fi Coordinator repository.

A release has seven assets: application APK, driver APK, application source ZIP,
driver corresponding-source archive, public signing key, `DISCLAIMER.txt`, and
`SHA256SUMS`. The manifest covers the other six files. Release notes identify the
target architecture and SHA256 of the public-key file. Check uploaded asset digests
against the local files. Generate the application source ZIP with `git archive`
from the exact release commit, using a `home-controller/` prefix. Source archives
remain snapshots of their release tags; later documentation corrections on `main`
do not change those tags or binaries.

For a documentation/package-only release, unchanged verified backend, radio helper
and driver artifacts may be reused. State that explicitly in the release notes;
do not present older radio or installation tests as newly performed checks. Build a
new application APK from the release source and version, verify its signature and
extracted contents, then generate the six-file checksum manifest. Packaging does not
itself install the APK or restart a running controller.

The installed interfaces and `/usr/share/home-controller/DISCLAIMER.txt` also expose
the notice. Current package hooks do not print it to the terminal during installation.

`.private/` holds local deployment settings and test evidence; `.tools/` holds local
development tools. Both are excluded from Git. Do not publish them, browser profiles,
cookies, private signing keys, TLS keys, control coordinates or radio addresses.
Generate release archives from tracked files, not by archiving the whole working
directory. A locally self-contained folder may contain private deployment material;
a public source archive must not.
