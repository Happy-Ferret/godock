// Copyright : (C) 2012-2016 by SQP
// E-mail    : sqp@glx-dock.org

/*
Gmail simple checker applet for the Cairo-Dock project.

Install

Install go and get go environment: you need a valid $GOPATH var and directory.

Download, build and install to your Cairo-Dock external applets dir:
  go get -d -u github.com/sqp/godock/applets/GoGmail  # download applet and dependencies.

  cd $GOPATH/src/github.com/sqp/godock/applets/GoGmail
  make        # compile the applet.
  make link   # link the applet to your external applet directory.

*/
package main

import (
	"github.com/sqp/godock/libs/appdbus"     // Connection to cairo-dock.
	"github.com/sqp/godock/services/GoGmail" // Applet service.
)

func main() { appdbus.StandAlone(GoGmail.NewApplet) }
