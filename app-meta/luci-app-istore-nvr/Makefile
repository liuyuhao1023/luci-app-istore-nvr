include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-istore-nvr
PKG_VERSION:=1.0.1
PKG_RELEASE:=1

PKG_LICENSE:=MIT
PKG_MAINTAINER:=liuyuhao1023

LUCI_TITLE:=LuCI support for iStore NVR Surveillance System
LUCI_DEPENDS:=+luci-base +mount-utils +cifs-utils +kmod-fs-cifs
LUCI_PKGARCH:=all

define Package/$(PKG_NAME)/conffiles
/etc/config/istore-nvr
endef

include $(TOPDIR)/feeds/luci/luci.mk

# call BuildPackage - OpenWrt buildroot signature
