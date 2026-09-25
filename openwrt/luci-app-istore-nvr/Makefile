include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-istore-nvr
PKG_VERSION:=1.0.1
PKG_RELEASE:=1

PKG_LICENSE:=MIT
PKG_LICENSE_FILES:=LICENSE
PKG_MAINTAINER:=liuyuhao1023

LUCI_TITLE:=LuCI support for iStore NVR Surveillance System
LUCI_DEPENDS:=+luci-base +mount-utils +cifs-utils +kmod-fs-cifs
LUCI_PKGARCH:=all

define Package/$(PKG_NAME)/conffiles
/etc/config/istore-nvr
endef

define Package/$(PKG_NAME)/postinst
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] || {
	chmod 755 /etc/init.d/istore-nvr 2>/dev/null || true
	chmod 755 /mnt/sata1-4/istore-nvr/istore-nvr 2>/dev/null || true
	chmod 755 /mnt/sata1-4/istore-nvr/mediamtx 2>/dev/null || true
	rm -f /tmp/luci-indexcache /tmp/luci-modulecache
	/etc/init.d/rpcd reload 2>/dev/null || true
	/etc/init.d/istore-nvr enable 2>/dev/null || true
}
exit 0
endef

define Package/$(PKG_NAME)/prerm
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] || {
	/etc/init.d/istore-nvr stop 2>/dev/null || true
	/etc/init.d/istore-nvr disable 2>/dev/null || true
}
exit 0
endef

include $(TOPDIR)/feeds/luci/luci.mk

# call BuildPackage - OpenWrt buildroot signature
