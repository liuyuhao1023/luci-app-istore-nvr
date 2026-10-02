include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-nvr-manager
PKG_VERSION:=1.0.1
PKG_RELEASE:=1
PKG_LICENSE:=MIT
PKG_MAINTAINER:=liuyuhao1023

LUCI_TITLE:=LuCI support for NVR Camera Surveillance System
LUCI_DESCRIPTION:=High-performance Network Video Recorder (NVR) surveillance system with multi-camera live view, direct streaming recording and NAS integration.
LUCI_DEPENDS:=+luci-base +mount-utils +cifs-utils +kmod-fs-cifs
LUCI_PKGARCH:=all

define Package/$(PKG_NAME)/conffiles
/etc/config/nvr-manager
endef

define Package/$(PKG_NAME)/postinst
#!/bin/sh
if [ -z "$${IPKG_INSTROOT}" ]; then
	chmod 755 /etc/init.d/nvr-manager 2>/dev/null || true
	/etc/init.d/rpcd reload 2>/dev/null || true
	rm -rf /tmp/luci-indexcache /tmp/luci-modulecache/
fi
exit 0
endef

define Package/$(PKG_NAME)/prerm
#!/bin/sh
if [ -z "$${IPKG_INSTROOT}" ]; then
	/etc/init.d/nvr-manager stop 2>/dev/null || true
	/etc/init.d/nvr-manager disable 2>/dev/null || true
fi
exit 0
endef

define Package/$(PKG_NAME)/postrm
#!/bin/sh
if [ -z "$${IPKG_INSTROOT}" ]; then
	/etc/init.d/rpcd reload 2>/dev/null || true
	rm -rf /tmp/luci-indexcache /tmp/luci-modulecache/
fi
exit 0
endef

include $(TOPDIR)/feeds/luci/luci.mk

# Support standalone build when feeds/luci/luci.mk is not directly present in out-of-tree env
$(eval $(call BuildPackage,$(PKG_NAME)))
