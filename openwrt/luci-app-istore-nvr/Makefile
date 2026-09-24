include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-istore-nvr
PKG_VERSION:=1.0.0
PKG_RELEASE:=1

PKG_LICENSE:=MIT
PKG_LICENSE_FILES:=LICENSE
PKG_MAINTAINER:=iStore NVR Team

LUCI_TITLE:=LuCI support for iStore NVR Surveillance System
LUCI_DEPENDS:=+luci-base +mount-utils +cifs-utils +kmod-fs-cifs
LUCI_PKGARCH:=all

include $(TOPDIR)/feeds/luci/luci.mk

# call BuildPackage - OpenWrt buildroot signature
