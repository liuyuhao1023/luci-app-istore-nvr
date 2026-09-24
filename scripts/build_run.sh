#!/bin/sh
set -e

# iStoreOS / OpenWrt 标准 .run 自解压安装包打包工具
PKG_NAME="luci-app-istore-nvr"
PKG_VERSION="1.0.0-1"
OUTPUT_DIR="$(pwd)/bin"
RUN_FILE="${OUTPUT_DIR}/${PKG_NAME}_${PKG_VERSION}_all.run"
TMP_DIR="/tmp/run_build_$$"

echo ">>> 开始构建 iStoreOS 原生 .run 安装包: ${RUN_FILE}"

rm -rf "$TMP_DIR"
mkdir -p "$TMP_DIR/payload" "$OUTPUT_DIR"

# 1. 拷贝 LuCI 插件文件
mkdir -p "$TMP_DIR/payload/etc/config"
mkdir -p "$TMP_DIR/payload/etc/init.d"
mkdir -p "$TMP_DIR/payload/usr/share/luci/menu.d"
mkdir -p "$TMP_DIR/payload/usr/share/rpcd/acl.d"
mkdir -p "$TMP_DIR/payload/www/luci-static/resources/view/istore-nvr"

cp openwrt/luci-app-istore-nvr/root/etc/config/* "$TMP_DIR/payload/etc/config/"
cp openwrt/luci-app-istore-nvr/root/etc/init.d/* "$TMP_DIR/payload/etc/init.d/"
cp openwrt/luci-app-istore-nvr/root/usr/share/luci/menu.d/* "$TMP_DIR/payload/usr/share/luci/menu.d/"
cp openwrt/luci-app-istore-nvr/root/usr/share/rpcd/acl.d/* "$TMP_DIR/payload/usr/share/rpcd/acl.d/"
cp openwrt/luci-app-istore-nvr/root/www/luci-static/resources/view/istore-nvr/* "$TMP_DIR/payload/www/luci-static/resources/view/istore-nvr/"

chmod 755 "$TMP_DIR/payload/etc/init.d/istore-nvr"

# 2. 生成自解压安装脚本 Header
cat << 'EOF' > "$TMP_DIR/installer.sh"
#!/bin/sh
# ==============================================================================
# OpenWrt NVR 摄像头管理系统 - iStoreOS 自动安装程序
# ==============================================================================
set -e

echo ">>> [1/4] 开始解压 NVR 摄像头管理系统组件..."
SKIP=$(awk '/^__ARCHIVE_BELOW__/ {print NR + 1; exit 0; }' "$0")
tail -n +$SKIP "$0" | tar -xz -C /

echo ">>> [2/4] 配置系统权限与服务注册..."
chmod 755 /etc/init.d/istore-nvr 2>/dev/null || true
chmod 755 /mnt/sata1-4/istore-nvr/istore-nvr 2>/dev/null || true
chmod 755 /mnt/sata1-4/istore-nvr/mediamtx 2>/dev/null || true

# 若备用目录存在核心程序，自动恢复到安装目录
if [ ! -f /mnt/sata1-4/istore-nvr/istore-nvr ] && [ -f /mnt/sata1-4/istore-nvr.backup/istore-nvr ]; then
    echo ">>> [恢复] 自动从备份目录恢复核心程序与数据..."
    mkdir -p /mnt/sata1-4/istore-nvr
    cp -rn /mnt/sata1-4/istore-nvr.backup/* /mnt/sata1-4/istore-nvr/ 2>/dev/null || true
    chmod 755 /mnt/sata1-4/istore-nvr/istore-nvr 2>/dev/null || true
    chmod 755 /mnt/sata1-4/istore-nvr/mediamtx 2>/dev/null || true
fi

echo ">>> [3/4] 清理 LuCI 缓存并重载 rpcd 权限..."
rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache* /usr/lib/lua/luci/controller/*nvr* /usr/lib/lua/luci/model/cbi/*nvr*
/etc/init.d/rpcd reload 2>/dev/null || true

echo ">>> [4/4] 启动服务并配置开机自启..."
/etc/init.d/istore-nvr enable 2>/dev/null || true
/etc/init.d/istore-nvr restart 2>/dev/null || true

echo "=========================================================="
echo " [✓] NVR 摄像头管理插件安装成功！"
echo " [✓] LuCI 页面：【服务】-> 【NVR摄像头管理】"
echo " [✓] 独立 Web 控制台：http://$(uci -q get network.lan.ipaddr || echo '192.168.1.15'):8080/"
echo "=========================================================="
exit 0
__ARCHIVE_BELOW__
EOF

# 3. 打包压缩 Payload 并拼接
tar -czf "$TMP_DIR/payload.tar.gz" -C "$TMP_DIR/payload" .
cat "$TMP_DIR/installer.sh" "$TMP_DIR/payload.tar.gz" > "$RUN_FILE"
chmod 755 "$RUN_FILE"

rm -rf "$TMP_DIR"
echo ">>> 成功生成 .run 安装包: $RUN_FILE"
