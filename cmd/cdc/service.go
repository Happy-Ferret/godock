package main

import (
	"github.com/sqp/godock/libs/srvdbus"
	"github.com/sqp/godock/services/allapps"

	"strings"
)

var cmdService = &Command{
	Run:       runService,
	UsageLine: "service [applet arguments list]",
	Short:     "start service with the dock or an applet",
	Long: `
Service handle the loading of the dock or its own packed applets.

Options:
  dock        Start the dock. This allow to relaunch the dock with its output
              in the same location.
  list        List available and active applets instances handled by the service.
  log         Open the service log terminal.
  stop        Stop the dock and the cdc service. (alias s k q quit)

The service option can also be called with options to start an applet. Those
options are provided by the dock when starting an applet. It only work for 
applets actually packed in this program.

Without arguments, the list will be displayed.

To enable the applet service for an applet, use a shell script in place of the
applet binary to forward the call:

  !/bin/sh 
  cdc service $* "$(pwd)" &
.`,
}

var gtkStart func()
var gtkStop func()

func runService(cmd *Command, args []string) {
	switch {
	case len(args) == 0 || args[0] == "list": // List active instances (default).
		clientSendLogged("service list", listServices, args)

	case args[0] == "k" || args[0] == "q" || args[0] == "s" || args[0] == "quit" || args[0] == "stop": // Stop the dock.
		clientSendLogged("service stop", stopDock, args)

	case args[0] == "d" || args[0] == "dock": // Start the service with cairo-dock.
		// if !logger.Err(srvdbus.StartDock(), "StartDock") {
		service(true, nil)
		// }

	case args[0] == "log": // Open the service log terminal.
		clientSendLogged("service log", logWindow, args)

	case len(args) == 6 || len(args) == 7: // Start applet. Need all arguments AFTER the command name.
		service(false, args)

	default:
		logger.Info("wrong arguments", strings.Join(args, " "))
	}
}

// Start Loader with the list of applets and args received for the first applet.
//
func service(withdock bool, args []string) {
	loader := srvdbus.NewLoader(allapps.List())
	active, e := loader.StartServer()
	if logger.Err(e, "StartServer") {
		return
	}

	if !active { // Someone else is active, forward the start applet.
		if len(args) > 0 {
			// Whether the first program instance will handle successfully the
			// request or not, this isn't our problem anymore, we still must quit.
			loader.Send("StartApplet", "", args[0], args[1], args[2], args[3], args[4], args[5], args[6])
		}
		return
	}

	if len(args) > 0 { // I am the chosen one. Let's create the first miracle.
		loader.StartApplet("", args[0], args[1], args[2], args[3], args[4], args[5], args[6])
	}

	if withdock { // Need to start a dock.
		if logger.Err(srvdbus.StartDock(), "StartDock") {
			return
		}
	}

	// Need to stay alive.

	// defer allapps.OnStop()
	if gtkStart != nil && allapps.GtkNeeded() {
		go func() { loader.StartLoop(withdock); logger.Info("cdc stopped"); gtkStop() }()
		gtkStart()

	} else {
		loader.StartLoop(withdock)
	}
}

func listServices(srv *srvdbus.Client, args []string) error {
	str, e := srv.ListServices()
	println(str)
	return e
}

func stopDock(srv *srvdbus.Client, args []string) error {
	return srv.StopDock()
}

func logWindow(srv *srvdbus.Client, args []string) error {
	return srv.LogWindow()
}
