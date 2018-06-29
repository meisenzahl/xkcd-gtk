################################################################################
# Build Variables
################################################################################

GO           = go
RM           = rm -f
MKDIR        = mkdir -p
INSTALL_EXE  = install
INSTALL_DATA = cp

BUILDFLAGS = -tags $(GTK_VERSION)
LDFLAGS    = -ldflags="-X main.appVersion=$(APP_VERSION)"

################################################################################
# Install Variables
################################################################################

prefix  = /usr
bindir  = $(prefix)/bin
datadir = $(prefix)/share

################################################################################
# Application Variables
################################################################################

APP = com.github.rkoesters.xkcd-gtk

EXE_NAME     = $(APP)
ICON_NAME    = $(APP).svg
DESKTOP_NAME = $(APP).desktop
APPDATA_NAME = $(APP).appdata.xml

EXE_PATH     = $(EXE_NAME)
ICON_PATH    = data/$(ICON_NAME)
DESKTOP_PATH = data/$(DESKTOP_NAME)
APPDATA_PATH = data/$(APPDATA_NAME)

################################################################################
# Automatic Variables
################################################################################

SOURCES = $(shell find . -type f -name '*.go')
DEPS    = $(shell tools/list-deps.sh ./...)

APP_VERSION = $(shell git describe --always --tags --dirty)
GTK_VERSION = $(shell tools/gtk-version.sh)

# If GOPATH isn't set, then just use the current directory.
ifeq "$(shell $(GO) env GOPATH)" ""
export GOPATH = $(shell pwd)
endif

################################################################################
# Targets
################################################################################

all: $(EXE_PATH)

deps:
	$(GO) get -u $(BUILDFLAGS) $(DEPS)

$(EXE_PATH): Makefile $(SOURCES)
	$(GO) build -o $@ $(BUILDFLAGS) $(LDFLAGS)

clean:
	$(RM) $(EXE_PATH)

install: $(EXE_PATH)
	$(MKDIR) $(DESTDIR)$(bindir)
	$(INSTALL_EXE) $(EXE_PATH) $(DESTDIR)$(bindir)
	$(MKDIR) $(DESTDIR)$(datadir)/icons/hicolor/scalable/apps
	$(INSTALL_DATA) $(ICON_PATH) $(DESTDIR)$(datadir)/icons/hicolor/scalable/apps
	$(MKDIR) $(DESTDIR)$(datadir)/applications
	$(INSTALL_DATA) $(DESKTOP_PATH) $(DESTDIR)$(datadir)/applications
	$(MKDIR) $(DESTDIR)$(datadir)/metainfo
	$(INSTALL_DATA) $(APPDATA_PATH) $(DESTDIR)$(datadir)/metainfo

uninstall:
	$(RM) $(DESTDIR)$(bindir)/$(EXE_NAME) \
	      $(DESTDIR)$(datadir)/icons/hicolor/scalable/apps/$(ICON_NAME) \
	      $(DESTDIR)$(datadir)/applications/$(DESKTOP_NAME) \
	      $(DESTDIR)$(datadir)/metainfo/$(APPDATA_NAME)

.PHONY: all clean deps install uninstall
