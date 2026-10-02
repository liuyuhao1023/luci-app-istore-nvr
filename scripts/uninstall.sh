#!/bin/sh
# ==============================================================================
# OpenWrt NVR (NVR 摄像头管理) 卸载清理脚本
# ==============================================================================

echo "=========================================================="
echo "  OpenWrt NVR 摄像头管理系统 - 卸载与清理向导            "
echo "=========================================================="

echo "[1/4] 停止并禁用开机自启..."
/etc/init.d/nvr-manager stop 2>/dev/null || true
/etc/init.d/nvr-manager disable 2>/dev/null || true

echo "[2/4] 删除服务启动脚本与 UCI 配置文件..."
rm -f /etc/init.d/nvr-manager
rm -f /etc/config/nvr-manager

echo "[3/4] 删除 LuCI 菜单注册、视图组件与 RPCD ACL 权限..."
rm -f /usr/share/luci/menu.d/luci-app-nvr-manager.json
rm -f /usr/share/rpcd/acl.d/luci-app-nvr-manager.json
rm -rf /www/luci-static/resources/view/nvr-manager

echo "[4/4] 清理 LuCI 菜单索引缓存并重载 rpcd..."
rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache*
/etc/init.d/rpcd reload 2>/dev/null || true

echo "=========================================================="
echo " [✓] 卸载与清理完成！LuCI 菜单及服务进程已安全移除。"
echo " 提示: 您的录像数据与数据库仍保存在外部磁盘中未被删除。"
echo "=========================================================="
