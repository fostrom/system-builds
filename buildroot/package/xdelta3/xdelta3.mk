################################################################################
#
# xdelta3
#
################################################################################

XDELTA3_VERSION = ff322e592383227b0d65ddfde7e0e5bbc504dc15
XDELTA3_SITE = $(call github,jmacd,xdelta,$(XDELTA3_VERSION))
XDELTA3_SUBDIR = xdelta3
XDELTA3_LICENSE = Apache-2.0
XDELTA3_LICENSE_FILES = xdelta3/LICENSE
XDELTA3_CPE_ID_VENDOR = xdelta_project
XDELTA3_CPE_ID_PRODUCT = xdelta
XDELTA3_CPE_ID_VERSION = 3.2.0

XDELTA3_CONF_OPTS = \
	-DXD3_ARMOR=OFF \
	-DXD3_BUILD_TESTS=OFF \
	-DXD3_BUILD_LIB=OFF \
	-DXD3_LZMA_MODE=off

$(eval $(cmake-package))
