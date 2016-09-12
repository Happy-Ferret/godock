TARGET=cdc
VERSION=0.0.3.5

SOURCE=github.com/sqp/godock

APPLETS=Audio Clouds Cpu DiskActivity DiskFree GoGmail Mem NetActivity Update


# unstable applets requires unmerged patches to build.
UNSTABLE=Notifications TVPlay
UNSTABLE_TAGS=gtk

# and dock even more, plus the rewritten dock.
DOCK=dock all

# Install prefix if any.
PKGDIR=

APPDIRGLDI=usr/share/cairo-dock/appletsgo/
APPDIRDBUS=usr/share/cairo-dock/plug-ins/Dbus/third-party/


# old version had:
# Could be useful for some distro packagers.
# FLAGSHARETHEME=$(SOURCE)/libs/gldi/maindock.CairoDockShareThemesDir '/usr/share/cairo-dock/themes'
# FLAGLOCALE=$(SOURCE)/libs/gldi/maindock.CairoDockLocaleDir '/usr/share/locale'

BUILDDATE=$(shell date --rfc-3339=seconds)

FLAGAPPVERSION='$(SOURCE)/libs/cdglobal.AppVersion=$(VERSION)'
FLAGGITHASH='$(SOURCE)/libs/cdglobal.GitHash=$(shell git rev-parse HEAD)'
# git describe --tags
FLAGBUILDDATE='$(SOURCE)/libs/cdglobal.BuildDate=$(BUILDDATE)'


FLAGS=-ldflags "-X $(FLAGAPPVERSION) -X $(FLAGBUILDDATE) -X $(FLAGGITHASH)"


%: build


build:
	go install -tags '$(APPLETS)'  $(FLAGS) $(SOURCE)/cmd/$(TARGET)

unstable:
	go install -tags '$(APPLETS) $(UNSTABLE) $(UNSTABLE_TAGS)' $(FLAGS) $(SOURCE)/cmd/$(TARGET)

dock:
	go install -tags '$(DOCK)' $(FLAGS) $(SOURCE)/cmd/$(TARGET)

patch:
	# Patch Dbus (for Notifications)
	cd "$(GOPATH)/src/github.com/godbus/dbus" && git pull --commit --no-edit https://github.com/sqp/dbus fixeavesdrop

	# Patch ini parser for config.
	cd "$(GOPATH)/src/github.com/go-ini/ini"  && git pull --commit --no-edit https://github.com/sqp/ini gtk_keyfile_compat

patch-dock: patch


install: install-common

	install -Dm755 "$(GOPATH)/src/$(SOURCE)/cmd/$(TARGET)/data/tocdc"  "$(PKGDIR)/usr/bin/tocdc"

	install -d "$(PKGDIR)/$(APPDIRDBUS)"
	for f in $(APPLETS); do	\
		cp -Rv --preserve=timestamps "applets/$$f"  "$(PKGDIR)/$(APPDIRDBUS)" ;\
		rm "$(PKGDIR)/$(APPDIRDBUS)/$$f/applet.go" ;\
		rm "$(PKGDIR)/$(APPDIRDBUS)/$$f/Makefile" ;\
		ln -s "/usr/bin/tocdc"  "$(PKGDIR)/$(APPDIRDBUS)/$$f/$$f" ;\
	done

install-dock: install-common

	install -d "$(PKGDIR)/$(APPDIRGLDI)"
	for f in $(APPLETS); do	\
		cp -Rv --preserve=timestamps "applets/$$f"  "$(PKGDIR)/$(APPDIRGLDI)" ;\
		rm $(PKGDIR)/$(APPDIRGLDI)/$$f/applet.go ;\
		rm $(PKGDIR)/$(APPDIRGLDI)/$$f/Makefile ;\
	done


install-common:
	install -p -Dm755 "$(GOPATH)/bin/$(TARGET)"  "$(PKGDIR)/usr/bin/$(TARGET)"

	install -Dm644 "$(GOPATH)/src/$(SOURCE)/LICENSE"  "$(PKGDIR)/usr/share/licenses/$(TARGET)/LICENSE"

	install -D "$(GOPATH)/src/$(SOURCE)/cmd/$(TARGET)/data/upload.nemo_action"  "$(PKGDIR)/usr/share/nemo/actions/cdc-upload.nemo_action"
	install -D "$(GOPATH)/src/$(SOURCE)/cmd/$(TARGET)/data/rework.conf"  "$(PKGDIR)/usr/share/cairo-dock/rework.conf"

	gzip -9 < "$(GOPATH)/src/$(SOURCE)/cmd/$(TARGET)/data/man.1" > "$(GOPATH)/src/$(SOURCE)/cmd/$(TARGET)/data/man.1.gz"
	install -pD "$(GOPATH)/src/$(SOURCE)/cmd/$(TARGET)/data/man.1.gz" "$(PKGDIR)/usr/share/man/man1/cdc.1.gz"


help:
	@# update command documentation.

	$(TARGET) help documentation > cmd/$(TARGET)/doc.go
	gofmt -w cmd/$(TARGET)/doc.go


stop:
	dbus-send --session --dest=org.cairodock.CairoDock /org/cdc/Cdc org.cairodock.CairoDock.Quit
	
	@## ActivateModule string:$(TARGET) boolean:false


cover:
	@# tests coverage with overalls: go get github.com/bluesuncorp/overalls

	overalls -covermode=count -debug  -project=$(SOURCE)
	go tool cover -html=overalls.coverprofile


# archive-%:
# 	go build -tags '$(APPLETS)'  -o applets/$(TARGET) $(SOURCE)/cmd/$(TARGET)
# 	@echo "Make archive $(TARGET)-$(VERSION)-$*.tar.xz"
# 	tar cJfv $(TARGET)-$(VERSION)-$*.tar.xz applets  --exclude-vcs
# 	rm applets/$(TARGET)
