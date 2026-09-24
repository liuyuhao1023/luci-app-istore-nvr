#!/bin/sh
set -e

# iStore NVR 标准 IPK 独立打包脚本 (遵循 iStoreOS 官方规范)
PKG_NAME="luci-app-istore-nvr"
PKG_VERSION="1.0.0-1"
PKG_ARCH="all"
BUILD_DIR="/tmp/ipk_build"
OUTPUT_DIR="$(pwd)/bin"

echo ">>> 开始构建 OpenWrt IPK 安装包: ${PKG_NAME}_${PKG_VERSION}_${PKG_ARCH}.ipk"

rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/data" "$BUILD_DIR/control" "$OUTPUT_DIR"

# 1. 拷贝数据文件
mkdir -p "$BUILD_DIR/data/etc/config"
mkdir -p "$BUILD_DIR/data/etc/init.d"
mkdir -p "$BUILD_DIR/data/usr/share/luci/menu.d"
mkdir -p "$BUILD_DIR/data/usr/share/rpcd/acl.d"
mkdir -p "$BUILD_DIR/data/www/luci-static/resources/view/istore-nvr"

cp -r openwrt/luci-app-istore-nvr/root/etc/config/* "$BUILD_DIR/data/etc/config/"
cp -r openwrt/luci-app-istore-nvr/root/etc/init.d/* "$BUILD_DIR/data/etc/init.d/"
cp -r openwrt/luci-app-istore-nvr/root/usr/share/luci/menu.d/* "$BUILD_DIR/data/usr/share/luci/menu.d/"
cp -r openwrt/luci-app-istore-nvr/root/usr/share/rpcd/acl.d/* "$BUILD_DIR/data/usr/share/rpcd/acl.d/"
cp -r openwrt/luci-app-istore-nvr/root/www/luci-static/resources/view/istore-nvr/* "$BUILD_DIR/data/www/luci-static/resources/view/istore-nvr/"

chmod +x "$BUILD_DIR/data/etc/init.d/istore-nvr"

# 2. 生成控制文件 (control)
cat << EOF > "$BUILD_DIR/control/control"
Package: ${PKG_NAME}
Version: ${PKG_VERSION}
Depends: libc, luci-base, mount-utils, cifs-utils
Section: luci
Architecture: ${PKG_ARCH}
Maintainer: liuyuhao1023
Description: LuCI support for OpenWrt NVR Camera Management System
EOF

# 3. 生成安装后触发脚本 (postinst - 遵守避坑指南: 显式权限修复与优雅服务重载)
cat << 'EOF' > "$BUILD_DIR/control/postinst"
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
EOF
chmod +x "$BUILD_DIR/control/postinst"

# 4. 生成卸载前脚本 (prerm)
cat << 'EOF' > "$BUILD_DIR/control/prerm"
#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] || {
	/etc/init.d/istore-nvr stop 2>/dev/null || true
	/etc/init.d/istore-nvr disable 2>/dev/null || true
}
exit 0
EOF
chmod +x "$BUILD_DIR/control/prerm"

# 5. 打包归档
echo "2.0" > "$BUILD_DIR/debian-binary"
tar -czf "$BUILD_DIR/data.tar.gz" -C "$BUILD_DIR/data" .
tar -czf "$BUILD_DIR/control.tar.gz" -C "$BUILD_DIR/control" .

IPK_FILE="${OUTPUT_DIR}/${PKG_NAME}_${PKG_VERSION}_${PKG_ARCH}.ipk"
tar -czf "$IPK_FILE" -C "$BUILD_DIR" debian-binary control.tar.gz data.tar.gz

rm -rf "$BUILD_DIR"
echo ">>> 成功生成 IPK 安装包: $IPK_FILE"
