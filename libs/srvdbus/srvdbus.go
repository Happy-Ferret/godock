// Package srvdbus provides a Dbus service (and client) with dock applets management.
package srvdbus

import (
	"github.com/godbus/dbus"

	"github.com/sqp/godock/libs/cdtype" // Logger type.
	"github.com/sqp/godock/libs/packages/versions"
	"github.com/sqp/godock/libs/srvdbus/dbuscommon" // Dbus service.

	"encoding/json"
	"strings"
	"time"
)

// SrvObj is the Dbus object name for the service.
const SrvObj = "org.cairodock.GoDock"

// SrvPath is the Dbus path name for the service.
const SrvPath = "/org/cairodock/GoDock"

// AppService defines common applets service actions to remotely interact with applets.
//
type AppService interface {
	CountActive() int
	GetApplets(name string) (list []cdtype.AppInstance)
	Tick()
}

// MgrDbus defines actions needed by the Dbus grouped applets manager.
//
type MgrDbus interface {
	IsActive(path string) bool
	OnSignal(path string, s *dbus.Signal) bool
}

// Loader is a multi applet manager.
//
type Loader struct {
	*dbuscommon.Server            // Dbus connection.
	apps               AppService // Applet actions (debug, upload).
	mgr                MgrDbus    // Applet activity forwarding (signals).
}

// NewLoader creates a loader with the given list of applets services.
//
func NewLoader(log cdtype.Logger) *Loader {
	srv := dbuscommon.NewServer(SrvObj, SrvPath, log)
	if srv == nil {
		return nil
	}
	return &Loader{Server: srv}
}

// SetManager sets the applet manager service.
//
func (load *Loader) SetManager(mgr AppService) {
	load.apps = mgr
	if db, ok := mgr.(MgrDbus); ok {
		load.mgr = db
	}
}

// Connect connects to the DBus API and starts the remote applets service.
//
func (load *Loader) Connect() (bool, error) {
	// var propsSpec = map[string]map[string]*prop.Prop{
	// 	SrvObj: {},
	// }

	return load.Start(load, nil)
}

//
//--------------------------------------------------------------------[ LOOP ]--

// StartLoop handle applets until there's none of them alive.
//
func (load *Loader) StartLoop() {
	defer load.Conn.ReleaseName(SrvObj)
	defer load.Log.Debug("Dbus service stopped")
	load.Log.Debug("Dbus service started")

	var waiter <-chan time.Time
	if load.apps != nil {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		waiter = ticker.C
	}

	for { // Main loop.

		select { // Wait for events, until the End signal is received from the dock.

		case s := <-load.Events: // Listen to DBus events.
			if load.dispatchDbusSignal(s) { // true if signal was Stop.

				// Keep service alive if we still manage some applets.
				if load.apps.CountActive() == 0 { // That's all folks. We're closing.
					return
				}
			}

		case <-waiter: // Tick every second to update pollers counters and launch actions.
			load.apps.Tick()
		}
	}
}

// Forward the Dbus signal to local manager or applet
//
func (load *Loader) dispatchDbusSignal(s *dbus.Signal) bool {
	path := strings.TrimSuffix(string(s.Path), "/sub_icons")

	switch {
	case s.Name == "org.freedesktop.DBus.NameAcquired": // Service started confirmed.

	case load.mgr != nil && load.mgr.IsActive(path): // Signal to applet.
		return load.mgr.OnSignal(path, s)

	default:
		load.Log.Info("unknown signal", s)

	}
	return false

}

//
//----------------------------------------------------------------[ DBUS API ]--

// UpToShareLastLink gets the link of the last item sent to a one-click hosting
// service.
//
func (load *Loader) UpToShareLastLink() (string, *dbus.Error) {
	var link string
	e := load.uploaderAction(func(app uploader) {
		link = app.UpToShareLastLink()
	})
	return link, e
}

// UpToShareLinks gets all links of items sent to one-click hosting services.
//
func (load *Loader) UpToShareLinks() (out []map[string]string, e *dbus.Error) {
	e = load.uploaderAction(func(app uploader) {
		out = app.UpToShareLinks()
	})
	return
}

// UploadString send raw text to a one-click hosting service.
//
func (load *Loader) UploadString(data string) *dbus.Error {
	return load.uploaderAction(func(app uploader) {
		app.UpToShareUploadString(data)
	})
}

// UploadFiles send files to a one-click hosting service.
//
func (load *Loader) UploadFiles(data []string) *dbus.Error {
	return load.uploaderAction(func(app uploader) {
		app.UploadFiles(data...)
	})
}

// SourceCodeVersions gets versions data.
// returned json: []*versions.Repo
//
func (load *Loader) SourceCodeVersions() (out []byte, err *dbus.Error) {
	err = load.sourceCoderAction(func(app sourceCoder) {
		var e error
		out, e = json.Marshal(app.Versions())
		load.Log.Err(e, "SourceCodeVersions marshal")
		// tmp := app.Versions()
		// out = make([]dbus.Variant, len(tmp))
		// for i, repo := range tmp {
		// 	println("add list", repo.Name)
		// 	out[i] = dbus.MakeVariant(repo)
		// }
	})
	return
}

// SourceCodeBuildTarget send data (raw text or file) to a one-click hosting service.
//
func (load *Loader) SourceCodeBuildTarget() *dbus.Error {
	return load.sourceCoderAction(func(app sourceCoder) {
		app.BuildTarget()
	})
}

// SourceCodeGrepTarget send data (raw text or file) to a one-click hosting service.
//
func (load *Loader) SourceCodeGrepTarget(data string) *dbus.Error {
	return load.sourceCoderAction(func(app sourceCoder) {
		app.GrepTarget(data)
	})
}

// SourceCodeOpenFile send data (raw text or file) to a one-click hosting service.
//
func (load *Loader) SourceCodeOpenFile(data string) *dbus.Error {
	return load.sourceCoderAction(func(app sourceCoder) {
		app.OpenFile(data)
	})
}

// AppletDebug change the debug state of an active applet.
//
func (load *Loader) AppletDebug(applet string, state bool) *dbus.Error {
	if load.apps == nil {
		return dbuscommon.NewError("no active application")
	}

	found := false
	for _, app := range load.apps.GetApplets(applet) {
		app.Log().SetDebug(state)
		found = true
	}
	if !found {
		load.Log.NewWarn("applet not found = "+applet, "set applet debug")
		return dbuscommon.NewError("applet not found = " + applet)
	}

	load.Log.Info("set applet debug", applet, state)
	return nil
}

//
//------------------------------------------------------[ DBUS SEND COMMANDS ]--

// AppletDebug forwards action set debug to a remote applet.
//
func AppletDebug(applet string, state bool) error {
	client, e := dbuscommon.GetClient(SrvObj, SrvPath)
	if e != nil {
		return e
	}
	return client.Call("AppletDebug", applet, state)
}

// SourceCodeBuildTarget forwards action build source target to the dock.
//
func SourceCodeBuildTarget() error {
	client, e := dbuscommon.GetClient(SrvObj, SrvPath)
	if e != nil {
		return e
	}
	return client.Call("SourceCodeBuildTarget")
}

// SourceCodeGrepTarget forwards action grep text in source code to the dock.
//
func SourceCodeGrepTarget(data string) error {
	client, e := dbuscommon.GetClient(SrvObj, SrvPath)
	if e != nil {
		return e
	}
	return client.Call("SourceCodeGrepTarget", data)
}

// SourceCodeOpenFile forwards action open source code file to the dock.
//
func SourceCodeOpenFile(data string) error {
	client, e := dbuscommon.GetClient(SrvObj, SrvPath)
	if e != nil {
		return e
	}
	return client.Call("SourceCodeOpenFile", data)
}

// SourceCodeVersions get source versions data.
//
func SourceCodeVersions() (list []*versions.Repo, e error) {
	client, e := dbuscommon.GetClient(SrvObj, SrvPath)
	if e != nil {
		return nil, e
	}

	var tmp []byte
	e = client.Get("SourceCodeVersions", []interface{}{&tmp})
	if e != nil {
		return nil, e
	}
	e = json.Unmarshal(tmp, &list)
	return list, e
}

// UploadString forwards action upload string content to the dock.
//
func UploadString(data string) error {
	client, e := dbuscommon.GetClient(SrvObj, SrvPath)
	if e != nil {
		return e
	}
	return client.Call("Upload", data)
}

// UploadFiles forwards action upload files (location) to the dock.
//
func UploadFiles(filePath ...string) error {
	client, e := dbuscommon.GetClient(SrvObj, SrvPath)
	if e != nil {
		return e
	}
	return client.Call("Upload", filePath)
}

// UpToShareLastLink get the last uploaded link from the dock.
//
func UpToShareLastLink() (string, error) {
	client, e := dbuscommon.GetClient(SrvObj, SrvPath)
	if e != nil {
		return "", e
	}

	var link string
	e = client.Get("UpToShareLastLink", []interface{}{&link})
	return link, e
}

// UpToShareLinks get all uploaded links from the dock.
//
func UpToShareLinks() (string, error) {
	client, e := dbuscommon.GetClient(SrvObj, SrvPath)
	if e != nil {
		return "", e
	}

	var link string
	e = client.Get("UpToShareLinks", []interface{}{&link})
	return link, e
}

//
//---------------------------------------------------------[ APPLETS OPTIONS ]--

type sourceCoder interface {
	BuildTarget() error
	GrepTarget(string)
	OpenFile(string)
	Versions() []*versions.Repo
}

func (load *Loader) sourceCoderAction(call func(sc sourceCoder)) *dbus.Error {
	if load.apps == nil {
		return dbuscommon.NewError("no active application")
	}

	uncasts := load.apps.GetApplets("Update")
	if len(uncasts) == 0 {
		return dbuscommon.NewError("no active sourceCoder found")
	}
	app := uncasts[0].(sourceCoder) // Send it to the first found. Should be safe for now, we can launch only one.
	call(app)
	return nil
}

type uploader interface {
	UploadFiles(...string)
	UpToShareUploadString(string)
	UpToShareLastLink() string
	UpToShareLinks() []map[string]string
}

func (load *Loader) uploaderAction(call func(sc uploader)) *dbus.Error {
	if load.apps == nil {
		return dbuscommon.NewError("no active application")
	}

	uncasts := load.apps.GetApplets("NetActivity")
	if len(uncasts) == 0 {
		return dbuscommon.NewError("no active uploader found")
	}
	app := uncasts[0].(uploader) // Send it to the first found. Should be safe for now, we can launch only one.
	call(app)
	return nil
}
