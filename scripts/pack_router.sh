#!/bin/sh
set -e
PKG_NAME="luci-app-istore-nvr"
PKG_VERSION="1.0.0-1"
PKG_ARCH="all"
BUILD_DIR="/tmp/ipk_build"

rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/data/etc/config"
mkdir -p "$BUILD_DIR/data/etc/init.d"
mkdir -p "$BUILD_DIR/data/usr/share/luci/menu.d"
mkdir -p "$BUILD_DIR/data/usr/share/rpcd/acl.d"
mkdir -p "$BUILD_DIR/data/www/luci-static/resources/view/istore-nvr"
mkdir -p "$BUILD_DIR/control"

cp /etc/config/istore-nvr "$BUILD_DIR/data/etc/config/"
cp /etc/init.d/istore-nvr "$BUILD_DIR/data/etc/init.d/"
cp /usr/share/luci/menu.d/luci-app-istore-nvr.json "$BUILD_DIR/data/usr/share/luci/menu.d/"
cp /usr/share/rpcd/acl.d/luci-app-istore-nvr.json "$BUILD_DIR/data/usr/share/rpcd/acl.d/"
cp /www/luci-static/resources/view/istore-nvr/* "$BUILD_DIR/data/www/luci-static/resources/view/istore-nvr/"

chmod 755 "$BUILD_DIR/data/etc/init.d/istore-nvr"

cat << 'CEOF' > "$BUILD_DIR/control/control"
Package: luci-app-istore-nvr
Version: 1.0.0-1
Depends: libc, luci-base, mount-utils, cifs-utils
Section: luci
Architecture: all
Maintainer: liuyuhao1023
Description: LuCI support for OpenWrt NVR Camera Management System
CEOF

cat << 'PEOF' > "$BUILD_DIR/control/postinst"
#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] || {
	chmod 755 /etc/init.d/istore-nvr 2>/dev/null || true
	chmod 755 /mnt/sata1-4/istore-nvr/istore-nvr 2>/dev/null || true
	chmod 755 /mnt/sata1-4/istore-nvr/mediamtx 2>/dev/null || true
	rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache* /usr/lib/lua/luci/controller/*nvr* /usr/lib/lua/luci/model/cbi/*nvr*
	/etc/init.d/rpcd reload 2>/dev/null || true
	/etc/init.d/istore-nvr enable 2>/dev/null || true
	/etc/init.d/istore-nvr restart 2>/dev/null || true
}
exit 0
PEOF
chmod +x "$BUILD_DIR/control/postinst"

cat << 'REOF' > "$BUILD_DIR/control/prerm"
#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] || {
	/etc/init.d/istore-nvr stop 2>/dev/null || true
	/etc/init.d/istore-nvr disable 2>/dev/null || true
}
exit 0
REOF
chmod +x "$BUILD_DIR/control/prerm"

echo "2.0" > "$BUILD_DIR/debian-binary"
tar -czf "$BUILD_DIR/data.tar.gz" -C "$BUILD_DIR/data" .
tar -czf "$BUILD_DIR/control.tar.gz" -C "$BUILD_DIR/control" .

tar -czf "/tmp/${PKG_NAME}_${PKG_VERSION}_${PKG_ARCH}.ipk" -C "$BUILD_DIR" debian-binary control.tar.gz data.tar.gz
rm -rf "$BUILD_DIR"
echo "Done: /tmp/${PKG_NAME}_${PKG_VERSION}_${PKG_ARCH}.ipk"
