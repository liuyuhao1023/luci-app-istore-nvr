include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-nvr-manager
PKG_VERSION:=1.0.1
PKG_RELEASE:=1

PKG_LICENSE:=MIT
PKG_LICENSE_FILES:=LICENSE
PKG_MAINTAINER:=liuyuhao1023

LUCI_TITLE:=LuCI support for NVR Camera Surveillance System
LUCI_DEPENDS:=+luci-base +mount-utils +cifs-utils +kmod-fs-cifs
LUCI_PKGARCH:=all

define Package/$(PKG_NAME)/conffiles
/etc/config/nvr-manager
endef

define Package/$(PKG_NAME)/postinst
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] || {
	chmod 755 /etc/init.d/nvr-manager 2>/dev/null || true
	chmod 755 /mnt/sata1-4/nvr-manager/nvr-manager 2>/dev/null || true
	chmod 755 /mnt/sata1-4/nvr-manager/mediamtx 2>/dev/null || true
	rm -f /tmp/luci-indexcache /tmp/luci-modulecache
	/etc/init.d/rpcd reload 2>/dev/null || true
	/etc/init.d/nvr-manager enable 2>/dev/null || true
}
exit 0
endef

define Package/$(PKG_NAME)/prerm
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] || {
	/etc/init.d/nvr-manager stop 2>/dev/null || true
	/etc/init.d/nvr-manager disable 2>/dev/null || true
}
exit 0
endef

include $(TOPDIR)/feeds/luci/luci.mk

# call BuildPackage - OpenWrt buildroot signature
