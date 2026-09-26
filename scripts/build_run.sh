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

# 2. 拷贝核心后端、前端最新 dist 与流媒体网关
mkdir -p "$TMP_DIR/payload/mnt/sata1-4/nvr-manager/dist"
mkdir -p "$TMP_DIR/payload/mnt/sata1-4/nvr-manager/data"
mkdir -p "$TMP_DIR/payload/mnt/sata1-4/recordings"

if [ -f "backend/istore-nvr" ]; then
    cp "backend/istore-nvr" "$TMP_DIR/payload/mnt/sata1-4/nvr-manager/nvr-manager"
    chmod 755 "$TMP_DIR/payload/mnt/sata1-4/nvr-manager/nvr-manager"
fi

if [ -d "backend/dist" ]; then
    cp -rf backend/dist/* "$TMP_DIR/payload/mnt/sata1-4/nvr-manager/dist/"
fi

if [ -f "mediamtx" ]; then
    cp "mediamtx" "$TMP_DIR/payload/mnt/sata1-4/nvr-manager/mediamtx"
    chmod 755 "$TMP_DIR/payload/mnt/sata1-4/nvr-manager/mediamtx"
fi

# 3. 生成自解压安装脚本 Header
cat << 'EOF' > "$TMP_DIR/installer.sh"
#!/bin/sh
# ==============================================================================
# OpenWrt NVR 视频监控管理系统 - 全自动一键部署程序
# ==============================================================================
set -e

echo ">>> [1/4] 开始解压 NVR 视频监控系统完整组件..."
SKIP=$(awk '/^__ARCHIVE_BELOW__/ {print NR + 1; exit 0; }' "$0")
tail -n +$SKIP "$0" | tar -xz -C /

echo ">>> [2/4] 配置系统权限与服务注册..."
chmod 755 /etc/init.d/nvr-manager 2>/dev/null || true
chmod 755 /mnt/sata1-4/nvr-manager/nvr-manager 2>/dev/null || true
chmod 755 /mnt/sata1-4/nvr-manager/mediamtx 2>/dev/null || true

echo ">>> [3/4] 清理 LuCI 缓存并重载 rpcd 权限..."
rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache* /usr/lib/lua/luci/controller/*nvr* /usr/lib/lua/luci/model/cbi/*nvr*
/etc/init.d/rpcd reload 2>/dev/null || true

echo ">>> [4/4] 启动服务并配置开机自启..."
/etc/init.d/nvr-manager enable 2>/dev/null || true
/etc/init.d/nvr-manager restart 2>/dev/null || true

sleep 2
if pidof nvr-manager >/dev/null; then
    echo "=========================================================="
    echo " [✓] NVR 视频监控管理系统部署成功！"
    echo " [✓] 核心服务 PID: $(pidof nvr-manager)"
    echo " [✓] LuCI 后台：【服务】-> 【NVR视频监控】"
    echo " [✓] 独立 Web 控制台：http://$(uci -q get network.lan.ipaddr || echo '192.168.1.15'):8080/"
    echo "=========================================================="
else
    echo "[-] 提示: 服务正在拉起中，请稍候刷新页面查看状态。"
fi
exit 0
__ARCHIVE_BELOW__
EOF

# 4. 打包压缩 Payload 并拼接
tar -czf "$TMP_DIR/payload.tar.gz" -C "$TMP_DIR/payload" .
cat "$TMP_DIR/installer.sh" "$TMP_DIR/payload.tar.gz" > "$RUN_FILE"
chmod 755 "$RUN_FILE"

rm -rf "$TMP_DIR"
echo ">>> 成功生成全功能自包含 .run 安装包: $RUN_FILE"
