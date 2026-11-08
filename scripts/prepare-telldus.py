import pathlib, sys, urllib.request, hashlib
p = pathlib.Path(sys.argv[1]) / 'Makefile'
s = p.read_text()
version = '2.1.3-beta1'
url = 'https://codeload.github.com/telldus/telldus/tar.gz/refs/tags/v' + version
data = urllib.request.urlopen(url).read()
s = s.replace('PKG_VERSION:=2.1.2', 'PKG_VERSION:=' + version)
s = s.replace('PKG_RELEASE:=9', 'PKG_RELEASE:=10')
s = s.replace('PKG_SOURCE:=$(PKG_NAME)-$(PKG_VERSION).tar.gz', 'PKG_SOURCE:=telldus-$(PKG_VERSION).tar.gz\nPKG_BUILD_DIR:=$(BUILD_DIR)/telldus-$(PKG_VERSION)/telldus-core')
s = s.replace('PKG_SOURCE_URL:=http://download.telldus.com/TellStick/Software/telldus-core/', 'PKG_SOURCE_URL:=' + url + '?')
import re
s = re.sub(r'PKG_HASH:=.*', 'PKG_HASH:=' + hashlib.sha256(data).hexdigest(), s)
p.write_text(s)
