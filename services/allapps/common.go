// Package allapps declares applets available for the applet loader service.
package allapps

import "github.com/sqp/godock/libs/dock"

// Common fields filled by declared applets.
var apps = make(map[string]func() dock.AppletInstance)
var needgtk bool // true if an applet has some GTK dependency.

// AddService is used to declare a service to the loader.
func AddService(name string, app func() dock.AppletInstance) {
	apps[name] = app
}

// List returns the list of declared applets.
//
func List() map[string]func() dock.AppletInstance {
	return apps
}

// AddGtkNeeded allow an applet to declare its gtk dependency.
// If used, the gtk main loop will lock the main thread to prevent later problems.
//
func AddGtkNeeded() {
	needgtk = true
}

// GtkNeeded returns true if an applet requires gtk.
func GtkNeeded() bool {
	return needgtk
}
