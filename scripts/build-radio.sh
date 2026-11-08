#!/bin/bash
set -euxo pipefail
ROOT=$(pwd)
SDK_VERSION=25.12.6
SDK_NAME=openwrt-sdk-${SDK_VERSION}-mvebu-cortexa9_gcc-14.3.0_musl_eabi.Linux-x86_64
BASE=https://downloads.openwrt.org/releases/$SDK_VERSION/targets/mvebu/cortexa9
mkdir -p build dist
cd build
python3 "$ROOT/scripts/download-sdk.py" "$BASE/$SDK_NAME.tar.zst" sdk.tar.zst
printf '%s  sdk.tar.zst\n' fc59dddc696b88e32662f4b1611c880bed24a88115ae423b640c81d071bec8ab | sha256sum -c -
tar --zstd -xf sdk.tar.zst
cd "$SDK_NAME"
./scripts/feeds update -a
./scripts/feeds install telldus-core
P=feeds/packages/utils/telldus-core
python3 "$ROOT/scripts/prepare-telldus.py" "$P"
cp "$ROOT/patches/1000-hasta-unit-zero.patch" "$P/patches/"
echo 'CONFIG_PACKAGE_telldus-core=m' >> .config
sed -i '/^CONFIG_AUTOREMOVE=/d' .config
echo '# CONFIG_AUTOREMOVE is not set' >> .config
make defconfig
make package/telldus-core/prepare V=s
SOURCE=$(find build_dir -maxdepth 2 -type d -name 'telldus-core-2.1.3_beta1' | head -1)
tar -czf "$ROOT/dist/telldus-core-2.1.3_beta1-source.tar.gz" -C "$(dirname "$SOURCE")" "$(basename "$SOURCE")" -C "$PWD/$P/.." telldus-core
make package/telldus-core/compile -j2 V=s
find bin -name 'telldus-core*.apk' -exec cp {} "$ROOT/dist/" \;
CC=$(find staging_dir/toolchain-* -path '*/bin/*-gcc' | head -1)
export STAGING_DIR="$PWD/staging_dir"
"$CC" -Os -s "$ROOT/src/radio.c" -ldl -o "$ROOT/dist/home-controller-radio"
LINK=$(find build_dir -path '*/telldus*/service/CMakeFiles/telldusd.dir/link.txt' | head -1)
python3 "$ROOT/scripts/build-radio-test.py" "$LINK" "$ROOT"
