#!/bin/sh
# ==============================================================================
# OpenWrt NVR (NVR 摄像头管理) 安装与部署脚本
# 适用平台: OpenWrt / iStoreOS / ImmortalWrt / 通用 Linux (x86_64 / aarch64)
# ==============================================================================

set -e

DATA_BASE_DIR="$1"
if [ -z "$DATA_BASE_DIR" ]; then
    echo "使用方法: $0 <外部存储目录>"
    echo "例如: $0 /mnt/sda1/nvr-manager"
    exit 1
fi

INSTALL_DIR="${DATA_BASE_DIR}/data"
RECORD_DIR="${DATA_BASE_DIR}/recordings"

echo "=========================================================="
echo "  OpenWrt NVR 摄像头管理系统 - 安装与配置向导            "
echo "=========================================================="

echo "[+] 核心数据与数据库目录: ${INSTALL_DIR}"
echo "[+] 本地切片录像存储目录: ${RECORD_DIR}"
mkdir -p "${INSTALL_DIR}" "${RECORD_DIR}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# 安装主服务可执行文件到 /usr/bin
if [ -f "${PROJECT_ROOT}/backend/istore-nvr" ]; then
    echo "[+] 部署后端主服务至 /usr/bin/nvr-manager..."
    cp -f "${PROJECT_ROOT}/backend/istore-nvr" "/usr/bin/nvr-manager"
    chmod 755 "/usr/bin/nvr-manager"
fi

# 安装静态前端至 /usr/share/nvr-manager/dist
if [ -d "${PROJECT_ROOT}/backend/dist" ]; then
    echo "[+] 部署 Web 前端静态资源至 /usr/share/nvr-manager/dist..."
    mkdir -p "/usr/share/nvr-manager/dist"
    cp -rf "${PROJECT_ROOT}/backend/dist/"* "/usr/share/nvr-manager/dist/"
fi

# 安装 mediamtx 至 /usr/bin
if [ -f "${PROJECT_ROOT}/mediamtx" ]; then
    echo "[+] 部署流媒体网关至 /usr/bin/mediamtx..."
    cp -f "${PROJECT_ROOT}/mediamtx" "/usr/bin/mediamtx"
    chmod 755 "/usr/bin/mediamtx"
fi

# 安装 LuCI 插件
if [ -d "${PROJECT_ROOT}/openwrt/luci-app-nvr-manager/root" ]; then
    echo "[+] 安装 LuCI 控制面板与系统服务配置..."
    cp -rf "${PROJECT_ROOT}/openwrt/luci-app-nvr-manager/root/etc/config/"* /etc/config/ 2>/dev/null || true
    cp -rf "${PROJECT_ROOT}/openwrt/luci-app-nvr-manager/root/etc/init.d/"* /etc/init.d/ 2>/dev/null || true
    cp -rf "${PROJECT_ROOT}/openwrt/luci-app-nvr-manager/root/usr/share/luci/menu.d/"* /usr/share/luci/menu.d/ 2>/dev/null || true
    cp -rf "${PROJECT_ROOT}/openwrt/luci-app-nvr-manager/root/usr/share/rpcd/acl.d/"* /usr/share/rpcd/acl.d/ 2>/dev/null || true
    mkdir -p /www/luci-static/resources/view/nvr-manager
    cp -rf "${PROJECT_ROOT}/openwrt/luci-app-nvr-manager/htdocs/luci-static/resources/view/nvr-manager/"* /www/luci-static/resources/view/nvr-manager/
fi

chmod 755 /etc/init.d/nvr-manager 2>/dev/null || true

# 配置 UCI
uci set nvr-manager.config.data_dir="${INSTALL_DIR}"
uci set nvr-manager.config.record_dir="${RECORD_DIR}"
uci commit nvr-manager

# 清理 LuCI 缓存
rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache*
/etc/init.d/rpcd reload 2>/dev/null || true

echo "=========================================================="
echo " [✓] 安装部署成功！请在 LuCI 中勾选启用服务后启动。"
echo "=========================================================="
