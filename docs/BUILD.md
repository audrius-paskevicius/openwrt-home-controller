# Build and publication

The repository is self-contained: application and UI source, radio helper, driver
patch, package scripts, tests, documentation and public package verification key are
included. External inputs are the Go toolchain, OpenWrt SDK/package sources and
Telldus upstream source. The initial release targets OpenWrt 25.12 on ARMv7
(mvebu/cortexa9); other targets require rebuilding the driver and radio helper.

On a Linux build host with Go, GCC build prerequisites, curl, tar and zstd:

```
go test -race ./...
node tests/check-js.cjs
node tests/account-ui.cjs
mkdir -p dist
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o dist/home-controller ./cmd/home-controller
bash scripts/build-radio.sh
```

The radio build uses the OpenWrt SDK and its package build definitions, upgrades
Telldus to 2.1.3-beta1, and applies the Hasta selflearningv2 unit-zero correction.
The source and patch must accompany distributed driver binaries.

Create your own signing key outside the public tree:

```
go run scripts/genkey.go /private/home-controller-signing.pem release/home-controller-public.pem
SIGN_KEY=/private/home-controller-signing.pem APK_TOOL=/path/to/apk.static sh scripts/build-apk.sh
```

Use apk-tools v3 with the `mkpkg` and `adbsign` applets. Re-sign the driver APK with
the same release key. Work on a copy: signing tools may modify a file even on failure.
The SDK uses a temporary key, so allow that input only during re-signing:

```
apk.static adbsign --allow-untrusted --reset-signatures --sign-key /private/home-controller-signing.pem dist/telldus-core-2.1.3_beta1-r10.apk
apk verify "dist/luci-app-home-controller-$(cat VERSION).apk" dist/telldus-core-2.1.3_beta1-r10.apk
```

Verification and installation must use the trusted public key, without
`--allow-untrusted`. Publish only sanitized release files. The radio build also
produces the patched corresponding-source archive and the `hasta-test` executable;
run that test on the target (it calculates packets without transmitting).
The public verification key belongs in `/etc/apk/keys/` on the target.

`node tests/browser-ui.cjs` exercises both interfaces in a disposable headless Chrome
listening on localhost:9222 (Node 24 or later). It injects synthetic API/RPC responses
and never contacts a router or sends radio commands. It checks mobile layouts,
control ordering, scheduler interaction, editors and inline save/error feedback.
Set `UI_SCREENSHOTS=1` to save household screenshots under ignored `.private/`.

`.private/` holds local deployment settings and test evidence; `.tools/` holds local
development tools. Both are excluded from Git. Do not publish them, browser profiles,
cookies, private signing keys, TLS keys, household coordinates or radio addresses.
Generate release archives from tracked files, not by archiving the whole working
directory. A locally self-contained folder may contain private deployment material;
a public source archive must not.
