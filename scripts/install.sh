#!/bin/sh
# ==============================================================================
# OpenWrt NVR (NVR 摄像头管理) 一键安装与部署脚本
# 适用平台: OpenWrt / iStoreOS / ImmortalWrt / 通用 Linux (x86_64 / aarch64)
# ==============================================================================

set -e

INSTALL_DIR="/mnt/sata1-4/istore-nvr"
RECORD_DIR="/mnt/sata1-4/recordings"

echo "=========================================================="
echo "  OpenWrt NVR 摄像头管理系统 - 快速安装与配置向导        "
echo "=========================================================="

# 1. 检查存储盘安全性 (严禁安装在软路由系统根分区 /overlay 上)
if [ ! -d "/mnt/sata1-4" ]; then
    echo "[-] 提示: 未检测到 /mnt/sata1-4 数据盘，请确保指定持久化外部存储盘。"
    INSTALL_DIR="/opt/istore-nvr"
    RECORD_DIR="/opt/recordings"
fi

echo "[+] 核心程序与数据库安装目录: ${INSTALL_DIR}"
echo "[+] 本地切片录像存储目录:     ${RECORD_DIR}"
mkdir -p "${INSTALL_DIR}/data" "${INSTALL_DIR}/dist" "${RECORD_DIR}"

# 2. 如果是从 git 仓库本地运行此脚本，同步核心文件
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [ -f "${PROJECT_ROOT}/backend/istore-nvr" ]; then
    echo "[+] 部署后端主服务二进制文件..."
    cp -f "${PROJECT_ROOT}/backend/istore-nvr" "${INSTALL_DIR}/istore-nvr"
    chmod 755 "${INSTALL_DIR}/istore-nvr"
fi

if [ -d "${PROJECT_ROOT}/backend/dist" ]; then
    echo "[+] 部署 Web 前端静态资源..."
    cp -rf "${PROJECT_ROOT}/backend/dist/"* "${INSTALL_DIR}/dist/"
fi

# 检查或复用现有 mediamtx 网关
if [ ! -f "${INSTALL_DIR}/mediamtx" ]; then
    if [ -f "/mnt/sata1-4/istore-nvr.backup/mediamtx" ]; then
        echo "[+] 复用备用目录中的 mediamtx 流媒体网关..."
        cp -f "/mnt/sata1-4/istore-nvr.backup/mediamtx" "${INSTALL_DIR}/mediamtx"
    elif [ -f "${PROJECT_ROOT}/mediamtx" ]; then
        cp -f "${PROJECT_ROOT}/mediamtx" "${INSTALL_DIR}/mediamtx"
    fi
fi
[ -f "${INSTALL_DIR}/mediamtx" ] && chmod 755 "${INSTALL_DIR}/mediamtx"

# 3. 安装 OpenWrt / LuCI 系统集成文件
if [ -d "${PROJECT_ROOT}/openwrt/luci-app-istore-nvr/root" ]; then
    echo "[+] 安装 LuCI 控制面板与系统服务配置..."
    cp -rf "${PROJECT_ROOT}/openwrt/luci-app-istore-nvr/root/etc/config/"* /etc/config/ 2>/dev/null || true
    cp -rf "${PROJECT_ROOT}/openwrt/luci-app-istore-nvr/root/etc/init.d/"* /etc/init.d/ 2>/dev/null || true
    cp -rf "${PROJECT_ROOT}/openwrt/luci-app-istore-nvr/root/usr/share/luci/menu.d/"* /usr/share/luci/menu.d/ 2>/dev/null || true
    cp -rf "${PROJECT_ROOT}/openwrt/luci-app-istore-nvr/root/usr/share/rpcd/acl.d/"* /usr/share/rpcd/acl.d/ 2>/dev/null || true
    mkdir -p /www/luci-static/resources/view/istore-nvr
    cp -rf "${PROJECT_ROOT}/openwrt/luci-app-istore-nvr/root/www/luci-static/resources/view/istore-nvr/"* /www/luci-static/resources/view/istore-nvr/
fi

chmod 755 /etc/init.d/istore-nvr 2>/dev/null || true

# 4. 刷新 LuCI 缓存与权限
echo "[+] 清理 LuCI 菜单缓存并重载服务..."
rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache* /usr/lib/lua/luci/controller/*nvr* /usr/lib/lua/luci/model/cbi/*nvr*
/etc/init.d/rpcd reload 2>/dev/null || true

# 5. 启动服务并设置自启
echo "[+] 启动 NVR 摄像头管理服务..."
/etc/init.d/istore-nvr enable
/etc/init.d/istore-nvr restart

sleep 2
if pidof istore-nvr >/dev/null; then
    echo "=========================================================="
    echo " [✓] 安装部署成功！"
    echo " [✓] 核心服务 PID: $(pidof istore-nvr)"
    echo " [✓] LuCI 菜单: 【服务】 -> 【NVR摄像头管理】"
    echo " [✓] 独立 Web 控制台: http://$(uci -q get network.lan.ipaddr || echo '192.168.1.15'):8080/"
    echo "=========================================================="
else
    echo "[-] 提示: 服务未能正常拉起，请查看系统日志: logread | grep istore-nvr"
fi
