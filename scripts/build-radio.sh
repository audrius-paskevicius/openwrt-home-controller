#!/bin/bash
set -euxo pipefail
ROOT=$(pwd)
SDK_VERSION=25.12.6
SDK_NAME=openwrt-sdk-${SDK_VERSION}-mvebu-cortexa9_gcc-14.3.0_musl_eabi.Linux-x86_64
BASE=https://downloads.openwrt.org/releases/$SDK_VERSION/targets/mvebu/cortexa9
mkdir -p build dist
cd build
python3 "$ROOT/scripts/download-sdk.py" "$BASE/$SDK_NAME.tar.zst" sdk.tar.zst
curl -fL --connect-timeout 20 --max-time 120 --retry 3 "$BASE/sha256sums" -o sums
grep " $SDK_NAME.tar.zst\$" sums | sed "s|$SDK_NAME.tar.zst|sdk.tar.zst|" | sha256sum -c -
tar --zstd -xf sdk.tar.zst
cd "$SDK_NAME"
./scripts/feeds update packages
./scripts/feeds install telldus-core
P=feeds/packages/utils/telldus-core
python3 "$ROOT/scripts/prepare-telldus.py" "$P"
cp "$ROOT/patches/1000-hasta-unit-zero.patch" "$P/patches/"
echo 'CONFIG_PACKAGE_telldus-core=m' >> .config
make defconfig
make package/telldus-core/compile -j2 V=s
find bin -name 'telldus-core*.apk' -exec cp {} "$ROOT/dist/" \;
CC=$(find staging_dir/toolchain-* -path '*/bin/*-gcc' | head -1)
export STAGING_DIR="$PWD/staging_dir"
"$CC" -Os -s "$ROOT/src/radio.c" -ldl -o "$ROOT/dist/home-controller-radio"
