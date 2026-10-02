#!/bin/sh
set -e

# OpenWrt / iStoreOS 标准 .run 自解压全功能独立安装包打包工具
PKG_NAME="luci-app-nvr-manager"
PKG_VERSION="1.0.1-1"
OUTPUT_DIR="$(pwd)/bin"
RUN_FILE="${OUTPUT_DIR}/${PKG_NAME}_${PKG_VERSION}_all.run"
TMP_DIR="/tmp/run_build_$$"

echo ">>> 开始构建全功能独立 .run 安装包: ${RUN_FILE}"

rm -rf "$TMP_DIR"
mkdir -p "$TMP_DIR/payload" "$OUTPUT_DIR"

# 1. 拷贝 LuCI 插件文件
mkdir -p "$TMP_DIR/payload/etc/config"
mkdir -p "$TMP_DIR/payload/etc/init.d"
mkdir -p "$TMP_DIR/payload/usr/share/luci/menu.d"
mkdir -p "$TMP_DIR/payload/usr/share/rpcd/acl.d"
mkdir -p "$TMP_DIR/payload/www/luci-static/resources/view/nvr-manager"

cp openwrt/luci-app-nvr-manager/root/etc/config/* "$TMP_DIR/payload/etc/config/"
cp openwrt/luci-app-nvr-manager/root/etc/init.d/* "$TMP_DIR/payload/etc/init.d/"
cp openwrt/luci-app-nvr-manager/root/usr/share/luci/menu.d/* "$TMP_DIR/payload/usr/share/luci/menu.d/"
cp openwrt/luci-app-nvr-manager/root/usr/share/rpcd/acl.d/* "$TMP_DIR/payload/usr/share/rpcd/acl.d/"
cp openwrt/luci-app-nvr-manager/htdocs/luci-static/resources/view/nvr-manager/* "$TMP_DIR/payload/www/luci-static/resources/view/nvr-manager/"

chmod 755 "$TMP_DIR/payload/etc/init.d/nvr-manager"

# 2. 拷贝核心后端、前端最新 dist 与流媒体网关到系统标准路径 (/usr/bin 与 /usr/share)
mkdir -p "$TMP_DIR/payload/usr/bin"
mkdir -p "$TMP_DIR/payload/usr/share/nvr-manager/dist"

if [ -f "backend/istore-nvr" ]; then
    cp "backend/istore-nvr" "$TMP_DIR/payload/usr/bin/nvr-manager"
    chmod 755 "$TMP_DIR/payload/usr/bin/nvr-manager"
fi

if [ -d "backend/dist" ]; then
    cp -rf backend/dist/* "$TMP_DIR/payload/usr/share/nvr-manager/dist/"
fi

if [ -f "mediamtx" ]; then
    cp "mediamtx" "$TMP_DIR/payload/usr/bin/mediamtx"
    chmod 755 "$TMP_DIR/payload/usr/bin/mediamtx"
fi

# 3. 生成自解压安装脚本 Header
cat << 'EOF' > "$TMP_DIR/installer.sh"
#!/bin/sh
# ==============================================================================
# OpenWrt NVR 视频监控管理系统 - 全自动一键部署程序
# ==============================================================================
set -e

echo ">>> [1/3] 开始解压 NVR 视频监控系统组件到系统标准路径..."
SKIP=$(awk '/^__ARCHIVE_BELOW__/ {print NR + 1; exit 0; }' "$0")
tail -n +$SKIP "$0" | tar -xz -C /

echo ">>> [2/3] 配置系统执行权限..."
chmod 755 /etc/init.d/nvr-manager 2>/dev/null || true
chmod 755 /usr/bin/nvr-manager 2>/dev/null || true
chmod 755 /usr/bin/mediamtx 2>/dev/null || true

echo ">>> [3/3] 清理 LuCI 缓存并重载 rpcd 权限..."
rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache*
/etc/init.d/rpcd reload 2>/dev/null || true

echo "=========================================================="
echo " [✓] NVR 视频监控管理系统安装成功！"
echo " [✓] LuCI 后台：【服务】-> 【NVR视频监控】"
echo " [!] 注意：请前往 LuCI 配置外部数据存储盘目录后启用服务。"
echo "=========================================================="

exit 0
__ARCHIVE_BELOW__
EOF

# 4. 打包压缩 Payload 并拼接
tar -czf "$TMP_DIR/payload.tar.gz" -C "$TMP_DIR/payload" .
cat "$TMP_DIR/installer.sh" "$TMP_DIR/payload.tar.gz" > "$RUN_FILE"
chmod 755 "$RUN_FILE"

rm -rf "$TMP_DIR"
echo ">>> 成功生成全功能自包含 .run 安装包: $RUN_FILE"
