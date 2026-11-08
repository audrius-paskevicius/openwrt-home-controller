import pathlib, sys, urllib.request, hashlib
p = pathlib.Path(sys.argv[1]) / 'Makefile'
s = p.read_text()
version = '2.1.3-beta1'
url = 'https://codeload.github.com/telldus/telldus/tar.gz/refs/tags/v' + version
data = urllib.request.urlopen(url).read()
digest = 'd40e87f29ebab4b18dc9d5334b2e21eb16976ed3b624f7f97e7e1be6eeabe53e'
if hashlib.sha256(data).hexdigest() != digest:
    raise RuntimeError('Telldus source checksum mismatch')
s = s.replace('PKG_VERSION:=2.1.2', 'PKG_VERSION:=2.1.3_beta1')
s = s.replace('PKG_RELEASE:=9', 'PKG_RELEASE:=10')
s = s.replace('PKG_NAME:=telldus-core', 'PKG_NAME:=telldus-core\nPKG_USE_NINJA:=0')
s = s.replace('PKG_SOURCE:=$(PKG_NAME)-$(PKG_VERSION).tar.gz', 'PKG_SOURCE:=telldus-2.1.3-beta1.tar.gz\nPKG_UNPACK:=$(HOST_TAR) -xzf $(DL_DIR)/$(PKG_SOURCE) --strip-components=2 -C $(PKG_BUILD_DIR) telldus-2.1.3-beta1/telldus-core')
s = s.replace('PKG_SOURCE_URL:=http://download.telldus.com/TellStick/Software/telldus-core/', 'PKG_SOURCE_URL:=' + url + '?')
import re
s = re.sub(r'PKG_HASH:=.*', 'PKG_HASH:=' + digest, s)
p.write_text(s)
# These warning fixes are already present in 2.1.3-beta1.
(p.parent / 'patches' / '110-fix_warnings.patch').unlink(missing_ok=True)
