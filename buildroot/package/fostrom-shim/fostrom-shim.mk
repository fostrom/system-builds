################################################################################
#
# fostrom-shim
#
################################################################################

FOSTROM_SHIM_VERSION = 1.38.0
FOSTROM_SHIM_SOURCE = busybox-$(FOSTROM_SHIM_VERSION).tar.bz2
FOSTROM_SHIM_SITE = https://www.busybox.net/downloads
FOSTROM_SHIM_LICENSE = GPL-2.0
FOSTROM_SHIM_LICENSE_FILES = LICENSE
FOSTROM_SHIM_CPE_ID_VENDOR = busybox
FOSTROM_SHIM_CPE_ID_PRODUCT = busybox

FOSTROM_SHIM_TARGET_PATH = /usr/libexec/fostrom/busybox

FOSTROM_SHIM_MAKE_OPTS = \
	AR="$(TARGET_AR)" \
	NM="$(TARGET_NM)" \
	RANLIB="$(TARGET_RANLIB)" \
	CC="$(TARGET_CC)" \
	ARCH=$(NORMALIZED_ARCH) \
	CROSS_COMPILE="$(TARGET_CROSS)" \
	SKIP_STRIP=y

define FOSTROM_SHIM_CONFIGURE_CMDS
	$(TARGET_MAKE_ENV) $(MAKE) -C $(@D) $(FOSTROM_SHIM_MAKE_OPTS) defconfig
	$(FOSTROM_SHIM_PKGDIR)/merge-config.sh $(@D)/.config \
		$(FOSTROM_SHIM_PKGDIR)/busybox.fragment
	$(TARGET_MAKE_ENV) $(MAKE) -C $(@D) $(FOSTROM_SHIM_MAKE_OPTS) oldconfig
endef

define FOSTROM_SHIM_BUILD_CMDS
	$(TARGET_MAKE_ENV) $(MAKE) -C $(@D) $(FOSTROM_SHIM_MAKE_OPTS) busybox
endef

define FOSTROM_SHIM_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/busybox \
		$(TARGET_DIR)$(FOSTROM_SHIM_TARGET_PATH)
endef

$(eval $(generic-package))
