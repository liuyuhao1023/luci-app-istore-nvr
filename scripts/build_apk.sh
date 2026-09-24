#!/bin/sh
set -e

# OpenWrt / Alpine APK 打包脚本 (遵循 APK 格式标准)
PKG_NAME="luci-app-istore-nvr"
PKG_VERSION="1.0.0-r1"
BUILD_DIR="/tmp/apk_build"
OUTPUT_DIR="$(pwd)/bin"

echo ">>> 开始构建 OpenWrt APK 安装包: ${PKG_NAME}-${PKG_VERSION}.apk"

rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/etc/config"
mkdir -p "$BUILD_DIR/etc/init.d"
mkdir -p "$BUILD_DIR/usr/share/luci/menu.d"
mkdir -p "$BUILD_DIR/usr/share/rpcd/acl.d"
mkdir -p "$BUILD_DIR/www/luci-static/resources/view/istore-nvr"
mkdir -p "$OUTPUT_DIR"

# 1. 拷贝文件
cp openwrt/luci-app-istore-nvr/root/etc/config/* "$BUILD_DIR/etc/config/"
cp openwrt/luci-app-istore-nvr/root/etc/init.d/* "$BUILD_DIR/etc/init.d/"
cp openwrt/luci-app-istore-nvr/root/usr/share/luci/menu.d/* "$BUILD_DIR/usr/share/luci/menu.d/"
cp openwrt/luci-app-istore-nvr/root/usr/share/rpcd/acl.d/* "$BUILD_DIR/usr/share/rpcd/acl.d/"
cp openwrt/luci-app-istore-nvr/root/www/luci-static/resources/view/istore-nvr/* "$BUILD_DIR/www/luci-static/resources/view/istore-nvr/"

chmod 755 "$BUILD_DIR/etc/init.d/istore-nvr"

# 2. 写入 .PKGINFO 元数据
cat << EOF > "$BUILD_DIR/.PKGINFO"
pkgname = ${PKG_NAME}
pkgver = ${PKG_VERSION}
pkgdesc = LuCI support for OpenWrt NVR Camera Management System
arch = all
origin = ${PKG_NAME}
maintainer = liuyuhao1023
license = MIT
depend = luci-base mount-utils cifs-utils
EOF

# 3. 写入 .post-install 触发脚本
cat << 'EOF' > "$BUILD_DIR/.post-install"
#!/bin/sh
chmod 755 /etc/init.d/istore-nvr 2>/dev/null || true
chmod 755 /mnt/sata1-4/istore-nvr/istore-nvr 2>/dev/null || true
chmod 755 /mnt/sata1-4/istore-nvr/mediamtx 2>/dev/null || true
rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache* /usr/lib/lua/luci/controller/*nvr* /usr/lib/lua/luci/model/cbi/*nvr*
/etc/init.d/rpcd reload 2>/dev/null || true
/etc/init.d/istore-nvr enable 2>/dev/null || true
/etc/init.d/istore-nvr restart 2>/dev/null || true
exit 0
EOF
chmod +x "$BUILD_DIR/.post-install"

# 4. 写入 .pre-deinstall 卸载触发脚本
cat << 'EOF' > "$BUILD_DIR/.pre-deinstall"
#!/bin/sh
/etc/init.d/istore-nvr stop 2>/dev/null || true
/etc/init.d/istore-nvr disable 2>/dev/null || true
exit 0
EOF
chmod +x "$BUILD_DIR/.pre-deinstall"

# 5. 打包归档为 .apk
APK_FILE="${OUTPUT_DIR}/${PKG_NAME}-${PKG_VERSION}.apk"
tar -czf "$APK_FILE" -C "$BUILD_DIR" .
rm -rf "$BUILD_DIR"
echo ">>> 成功生成 APK 安装包: $APK_FILE"
