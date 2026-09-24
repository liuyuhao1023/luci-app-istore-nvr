#!/bin/sh
set -e

# iStore NVR 双格式 (IPK & APK) 规范打包脚本
# 遵循 iStoreOS 官方商店规范与避坑指南标准
PKG_NAME="luci-app-istore-nvr"
PKG_VERSION="1.0.0"
PKG_RELEASE="1"
PKG_ARCH="all"
BUILD_DIR="/tmp/istore_nvr_pkg_build"
OUTPUT_DIR="$(pwd)/bin"

echo "=========================================================="
echo ">>> 开始构建 iStoreOS 规范软件包: ${PKG_NAME} v${PKG_VERSION}-${PKG_RELEASE}"
echo "=========================================================="

rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/data" "$BUILD_DIR/control" "$OUTPUT_DIR"

# 1. 组装数据目录结构
mkdir -p "$BUILD_DIR/data/etc/config"
mkdir -p "$BUILD_DIR/data/etc/init.d"
mkdir -p "$BUILD_DIR/data/usr/lib/lua/luci/controller"
mkdir -p "$BUILD_DIR/data/usr/lib/lua/luci/model/cbi"
mkdir -p "$BUILD_DIR/data/usr/share/rpcd/acl.d"

cp -r openwrt/luci-app-istore-nvr/root/etc/config/* "$BUILD_DIR/data/etc/config/"
cp -r openwrt/luci-app-istore-nvr/root/etc/init.d/* "$BUILD_DIR/data/etc/init.d/"
cp -r openwrt/luci-app-istore-nvr/luasrc/controller/* "$BUILD_DIR/data/usr/lib/lua/luci/controller/"
cp -r openwrt/luci-app-istore-nvr/luasrc/model/cbi/* "$BUILD_DIR/data/usr/lib/lua/luci/model/cbi/"
cp -r openwrt/luci-app-istore-nvr/root/usr/share/rpcd/acl.d/* "$BUILD_DIR/data/usr/share/rpcd/acl.d/"

# 2. 生成控制文件 (control)
cat << EOF > "$BUILD_DIR/control/control"
Package: ${PKG_NAME}
Version: ${PKG_VERSION}-${PKG_RELEASE}
Depends: libc, luci-base, mount-utils, cifs-utils
Section: luci
Architecture: ${PKG_ARCH}
Maintainer: iStore NVR Team
Description: LuCI support for iStore NVR Surveillance System
EOF

# 3. 安装后脚本 (postinst) - 严格遵循避坑指南：显式自愈权限与标准 rpcd reload
cat << 'EOF' > "$BUILD_DIR/control/postinst"
#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] || {
	# 核心避坑 1：显式兜底赋予权限，彻底解决 Windows 开发权限丢失
	chmod 755 /etc/init.d/istore-nvr 2>/dev/null || true
	chmod 755 /mnt/sata1-4/istore-nvr/istore-nvr 2>/dev/null || true
	chmod 755 /mnt/sata1-4/istore-nvr/mediamtx 2>/dev/null || true
	
	# 刷新 LuCI 缓存
	rm -f /tmp/luci-indexcache /tmp/luci-modulecache
	
	# 核心避坑 2：严禁 killall -HUP rpcd，必须使用标准 service reload
	/etc/init.d/rpcd reload 2>/dev/null || true
	
	# 启用并自启服务
	/etc/init.d/istore-nvr enable 2>/dev/null || true
	/etc/init.d/istore-nvr restart 2>/dev/null || true
}
exit 0
EOF
chmod +x "$BUILD_DIR/control/postinst"

# 4. 卸载前脚本 (prerm)
cat << 'EOF' > "$BUILD_DIR/control/prerm"
#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] || {
	/etc/init.d/istore-nvr stop 2>/dev/null || true
	/etc/init.d/istore-nvr disable 2>/dev/null || true
}
exit 0
EOF
chmod +x "$BUILD_DIR/control/prerm"

# 5. 打包为 opkg (.ipk) 格式 (兼容 OpenWrt 21.x ~ 23.x / iStoreOS 现行版本)
echo "2.0" > "$BUILD_DIR/debian-binary"
tar -czf "$BUILD_DIR/data.tar.gz" -C "$BUILD_DIR/data" .
tar -czf "$BUILD_DIR/control.tar.gz" -C "$BUILD_DIR/control" .

IPK_FILE="${OUTPUT_DIR}/${PKG_NAME}_${PKG_VERSION}-${PKG_RELEASE}_${PKG_ARCH}.ipk"
tar -czf "$IPK_FILE" -C "$BUILD_DIR" debian-binary control.tar.gz data.tar.gz
echo ">>> [1/2] 成功生成 IPK 安装包: $IPK_FILE"

# 6. 打包为 apk 格式 (兼容 OpenWrt 24.x 现代版本)
# APK 包封装规范：gzip 压缩格式包含 .PKGINFO 与对应文件树
mkdir -p "$BUILD_DIR/apk_root"
cat << EOF > "$BUILD_DIR/apk_root/.PKGINFO"
pkgname = ${PKG_NAME}
pkgver = ${PKG_VERSION}-r${PKG_RELEASE}
pkgdesc = LuCI support for iStore NVR Surveillance System
arch = ${PKG_ARCH}
origin = ${PKG_NAME}
maintainer = iStore NVR Team
license = MIT
depend = luci-base mount-utils cifs-utils
EOF
cp -r "$BUILD_DIR/data/"* "$BUILD_DIR/apk_root/"
APK_FILE="${OUTPUT_DIR}/${PKG_NAME}-${PKG_VERSION}-r${PKG_RELEASE}.apk"
tar -czf "$APK_FILE" -C "$BUILD_DIR/apk_root" .
echo ">>> [2/2] 成功生成 APK 安装包: $APK_FILE"

rm -rf "$BUILD_DIR"
echo "=========================================================="
echo ">>> 双格式打包全部完成！产物输出于 bin/ 目录"
echo "=========================================================="
