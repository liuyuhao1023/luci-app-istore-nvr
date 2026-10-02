#!/bin/sh
set -e

# iStore NVR 标准 IPK 独立打包脚本 (遵循 iStoreOS 官方规范)
PKG_NAME="luci-app-nvr-manager"
PKG_VERSION="1.0.1-1"
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
mkdir -p "$BUILD_DIR/data/www/luci-static/resources/view/nvr-manager"

cp -r openwrt/luci-app-nvr-manager/root/etc/config/* "$BUILD_DIR/data/etc/config/"
cp -r openwrt/luci-app-nvr-manager/root/etc/init.d/* "$BUILD_DIR/data/etc/init.d/"
cp -r openwrt/luci-app-nvr-manager/root/usr/share/luci/menu.d/* "$BUILD_DIR/data/usr/share/luci/menu.d/"
cp -r openwrt/luci-app-nvr-manager/root/usr/share/rpcd/acl.d/* "$BUILD_DIR/data/usr/share/rpcd/acl.d/"
cp -r openwrt/luci-app-nvr-manager/htdocs/luci-static/resources/view/nvr-manager/* "$BUILD_DIR/data/www/luci-static/resources/view/nvr-manager/"

chmod 755 "$BUILD_DIR/data/etc/init.d/nvr-manager"

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

# 3. 生成安装后触发脚本 (postinst - 遵守 iStoreOS 规范: 权限修复与缓存清理，不默认启动)
cat << 'EOF' > "$BUILD_DIR/control/postinst"
#!/bin/sh
if [ -z "${IPKG_INSTROOT}" ]; then
	chmod 755 /etc/init.d/nvr-manager 2>/dev/null || true
	/etc/init.d/rpcd reload 2>/dev/null || true
	rm -rf /tmp/luci-indexcache /tmp/luci-modulecache/
fi
exit 0
EOF
chmod +x "$BUILD_DIR/control/postinst"

# 4. 生成卸载前脚本 (prerm)
cat << 'EOF' > "$BUILD_DIR/control/prerm"
#!/bin/sh
if [ -z "${IPKG_INSTROOT}" ]; then
	/etc/init.d/nvr-manager stop 2>/dev/null || true
	/etc/init.d/nvr-manager disable 2>/dev/null || true
fi
exit 0
EOF
chmod +x "$BUILD_DIR/control/prerm"

# 5. 生成卸载后脚本 (postrm)
cat << 'EOF' > "$BUILD_DIR/control/postrm"
#!/bin/sh
if [ -z "${IPKG_INSTROOT}" ]; then
	/etc/init.d/rpcd reload 2>/dev/null || true
	rm -rf /tmp/luci-indexcache /tmp/luci-modulecache/
fi
exit 0
EOF
chmod +x "$BUILD_DIR/control/postrm"

# 6. 打包归档 (严格使用 ./ 相对路径前缀，符合 iStoreOS CI 检验规范)
echo "2.0" > "$BUILD_DIR/debian-binary"
tar -czf "$BUILD_DIR/data.tar.gz" -C "$BUILD_DIR/data" .
tar -czf "$BUILD_DIR/control.tar.gz" -C "$BUILD_DIR/control" .

IPK_FILE="${OUTPUT_DIR}/${PKG_NAME}_1.0.1_${PKG_ARCH}.ipk"
cd "$BUILD_DIR"
tar -czf "$IPK_FILE" ./debian-binary ./control.tar.gz ./data.tar.gz
cd - >/dev/null

rm -rf "$BUILD_DIR"
echo ">>> 成功生成 IPK 安装包: $IPK_FILE"
