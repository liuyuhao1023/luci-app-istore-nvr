#!/bin/sh
# ==============================================================================
# OpenWrt NVR (NVR 摄像头管理) 卸载清理脚本
# ==============================================================================

echo "=========================================================="
echo "  OpenWrt NVR 摄像头管理系统 - 卸载与清理向导            "
echo "=========================================================="

echo "[1/4] 停止并禁用开机自启..."
/etc/init.d/istore-nvr stop 2>/dev/null || true
/etc/init.d/istore-nvr disable 2>/dev/null || true
killall -9 istore-nvr mediamtx 2>/dev/null || true

echo "[2/4] 删除服务启动脚本与 UCI 配置文件..."
rm -f /etc/init.d/istore-nvr
rm -f /etc/config/istore-nvr

echo "[3/4] 删除 LuCI 菜单注册、视图组件与 RPCD ACL 权限..."
rm -f /usr/share/luci/menu.d/luci-app-istore-nvr.json
rm -f /usr/share/rpcd/acl.d/luci-app-istore-nvr.json
rm -rf /www/luci-static/resources/view/istore-nvr
rm -f /usr/lib/lua/luci/controller/*nvr*
rm -f /usr/lib/lua/luci/model/cbi/*nvr*

echo "[4/4] 清理 LuCI 菜单索引缓存并重载 rpcd..."
rm -rf /tmp/luci-indexcache* /tmp/luci-modulecache* /tmp/ipk_build /tmp/*.ipk
/etc/init.d/rpcd reload 2>/dev/null || true

echo "=========================================================="
echo " [✓] 卸载与清理完成！LuCI 菜单及服务进程已全部移除。"
echo " 提示: 您的录像数据与数据库仍保存在 /mnt/sata1-4 中未被删除。"
echo "=========================================================="
