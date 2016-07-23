// Package dlogbus provides a Dbus service (and client) for a dock external launcher.
package dlogbus

import (
	"github.com/godbus/dbus"
	"github.com/godbus/dbus/introspect"

	"github.com/sqp/godock/libs/cdtype"             // Logger type.
	"github.com/sqp/godock/libs/srvdbus/dbuscommon" // Dbus base object.
	"github.com/sqp/godock/libs/srvdbus/dockbus"    // Send dock restart.
	"github.com/sqp/godock/libs/srvdbus/dockpath"   // Path to main dock dbus service.
)

// SrvObj is the Dbus object name for the service.
const SrvObj = "org.cairodock.DockLog"

// SrvPath is the Dbus path name for the service.
const SrvPath = "/org/cairodock/DockLog"

// Introspec is the Dbus introspect text with service methods.
const Introspec = `<node>
	<interface name="` + SrvObj + `">
		<method name="Restart">
		</method>
	</interface>` +
	introspect.IntrospectDataString + `
</node> `

//
//------------------------------------------------------------------[ CLIENT ]--

// Client defines a Dbus client to connect to the dlogbus server.
//
type Client struct {
	*dbuscommon.Client
}

// Action sends an action to the dlogbus server.
//
func Action(action func(*Client) error) error {
	client, e := dbuscommon.GetClient(SrvObj, SrvPath)
	if e != nil {
		return e
	}
	return action(&Client{client}) // we have a server, launch the provided action.
}

// Restart sends the Restart action to the dlogbus server.
//
func (client *Client) Restart() error {
	return client.Call("Restart")
}

//
//------------------------------------------------------------------[ SERVER ]--

// Server defines a Dbus server that manage the state of a cdc program.
//
type Server struct {
	*dbuscommon.Server // Dbus connection.

	DockArgs []string
	Started  bool

	needRestart bool
	over        chan struct{}
}

// NewServer creates a dlogbus server instance with cdc command args.
// Only one can be active.
//
func NewServer(dockArgs []string, log cdtype.Logger) *Server {
	return &Server{
		Server:   dbuscommon.NewServer(SrvObj, SrvPath, log),
		DockArgs: dockArgs,
		over:     make(chan struct{}),
	}
}

// SetArgs sets the dock command args.
//
func (o *Server) SetArgs(args []string) *Server {
	o.DockArgs = args
	return o
}

// DockStart starts the dock.
//
func (o *Server) DockStart() error {

	if o.IsStarted() { // && !o.needRestart { // on restart, the dock is too slow to quit, but can be relaunched early.
		return nil
	}

	cmd := o.Log.ExecCmd("cdc", o.DockArgs...)
	e := cmd.Start()
	if e != nil {
		return e
	}

	go func() {
		o.Started = true

		cmd.Wait()

		o.Started = false

		if o.needRestart {
			o.needRestart = false
			o.over <- struct{}{}
		}
	}()
	return nil
}

// DockStop stops the dock.
//
func (o *Server) DockStop() error {
	dockpath.DbusPathDock = "/org/cdc/Cdc"
	return dockbus.Send(dockbus.DockQuit)
}

// DockRestart restarts the dock.
//
func (o *Server) DockRestart() error {

	if o.IsStarted() {
		e := o.DockStop()
		o.Log.Err(e, "StopDock")
	}

	// Wait dock quit
	o.needRestart = true

	<-o.over

	return o.DockStart()
}

// IsStarted returns whether the managed dock is started or not.
//
func (o *Server) IsStarted() bool {
	return o.Started
}

//
//----------------------------------------------------------------[ DBUS API ]--

// Restart restarts the dock.
//
func (o *Server) Restart() *dbus.Error {
	o.DockRestart()
	return nil
}
