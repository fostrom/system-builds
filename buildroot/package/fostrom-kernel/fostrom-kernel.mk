################################################################################
#
# fostrom-kernel
#
################################################################################

FOSTROM_KERNEL_VERSION = 0.1.0
FOSTROM_KERNEL_TARGET = $(call qstrip,$(BR2_PACKAGE_FOSTROM_KERNEL_TARGET))
FOSTROM_KERNEL_SOURCE = kernel-$(FOSTROM_KERNEL_TARGET)-$(FOSTROM_KERNEL_VERSION).tar.zst
FOSTROM_KERNEL_SITE = https://github.com/fostrom/kernel-builds/releases/download/v$(FOSTROM_KERNEL_VERSION)
FOSTROM_KERNEL_STRIP_COMPONENTS = 0
FOSTROM_KERNEL_LICENSE = GPL-2.0 WITH Linux-syscall-note, firmware under its own licenses
FOSTROM_KERNEL_LICENSE_FILES = usr/share/licenses/linux/COPYING usr/share/licenses/linux-firmware/WHENCE
FOSTROM_KERNEL_INSTALL_IMAGES = YES

# Modules, firmware, licences and module options, and on GRUB targets the kernel
# and microcode
define FOSTROM_KERNEL_INSTALL_TARGET_CMDS
	mkdir -p $(TARGET_DIR)/lib $(TARGET_DIR)/usr/share
	cp -dpR $(@D)/lib/. $(TARGET_DIR)/lib/
	cp -dpR $(@D)/usr/share/licenses $(TARGET_DIR)/usr/share/
	if [ -d $(@D)/etc ]; then cp -dpR $(@D)/etc/. $(TARGET_DIR)/etc/; fi
	$(if $(BR2_TARGET_GRUB2),cp -dpR $(@D)/boot $(TARGET_DIR)/)
endef

# The kernel config, and on the Pi the kernel and device trees
define FOSTROM_KERNEL_INSTALL_IMAGES_CMDS
	$(INSTALL) -D -m 0644 $(@D)/kernel.config $(BINARIES_DIR)/kernel.config
	$(if $(BR2_TARGET_GRUB2),,cp -dpR $(@D)/boot/. $(BINARIES_DIR)/)
endef

$(eval $(generic-package))
