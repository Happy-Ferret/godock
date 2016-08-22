// Package DiskActivity is a monitoring applet for Cairo-Dock.
package DiskActivity

import (
	"github.com/sqp/godock/libs/cdapplet" // Applet base.
	"github.com/sqp/godock/libs/cdtype"   // Applet types.
	"github.com/sqp/godock/libs/sysinfo"
	"github.com/sqp/godock/libs/text/bytesize"

	"fmt"
)

//
//------------------------------------------------------------------[ APPLET ]--

// Applet data and controlers.
//
type Applet struct {
	cdtype.AppBase // Applet base and dock connection.

	conf    *appletConf
	service *sysinfo.IOActivity
}

// NewApplet create a new applet instance.
//
func NewApplet() cdtype.AppInstance {
	app := &Applet{}
	app.AppBase = cdapplet.New(&app.conf) // Icon controler and interface to cairo-dock.

	app.service = sysinfo.NewIOActivity(app)
	app.service.Log = app.Log()
	app.service.FormatIcon = formatIcon
	app.service.FormatLabel = formatLabel
	app.service.GetData = sysinfo.GetDiskActivity

	app.Poller().Add(app.service.Check)

	return app
}

// Init load user configuration if needed and initialise applet.
//
func (app *Applet) Init(def *cdtype.Defaults, confLoaded bool) {
	// Settings for poller and IOActivity (force renderer reset in case of reload).
	app.service.Settings(uint64(app.conf.UpdateDelay.Value()), cdtype.InfoPosition(app.conf.DisplayText), app.conf.DisplayValues, app.conf.GraphType, app.conf.GaugeName, app.conf.Disks...)

	// Defaults.
	def.PollerInterval = app.conf.UpdateDelay.Value()
	def.Commands = cdtype.Commands{
		cmdLeft:   cdtype.NewCommandStd(app.conf.LeftAction, app.conf.LeftCommand, app.conf.LeftClass),
		cmdMiddle: cdtype.NewCommandStd(app.conf.MiddleAction, app.conf.MiddleCommand),
	}
}

//
//------------------------------------------------------------------[ EVENTS ]--

// OnClick launch the configured action on user click.
//
func (app *Applet) OnClick(int) {
	app.Command().Launch(cmdLeft)
}

// OnMiddleClick launch the configured action on user middle click.
//
func (app *Applet) OnMiddleClick() {
	app.Command().Launch(cmdMiddle)
}

// OnBuildMenu fills the menu with left and middle click actions if they're set.
//
func (app *Applet) OnBuildMenu(menu cdtype.Menuer) {
	if app.conf.LeftAction > 0 && app.conf.LeftCommand != "" {
		menu.AddEntry("Action left click", "system-run", app.OnClick)
	}
	if app.conf.MiddleAction > 0 && app.conf.MiddleCommand != "" {
		menu.AddEntry("Action middle click", "system-run", app.OnMiddleClick)
	}
}

//
//-----------------------------------------------------------------[ DISPLAY ]--

// Quick-info display callback. One line for each value. Zero are replaced by empty string.
//
func formatIcon(dev string, in, out uint64) string {
	return sysinfo.FormatRate(in*BlockSize) + "\n" + sysinfo.FormatRate(out*BlockSize)
}

// Label display callback. One line for each device. Format="eth0: r 42 / w 128".
//
func formatLabel(dev string, in, out uint64) string {
	return fmt.Sprintf("%s: %s %s / %s %s", dev, "r", bytesize.ByteSize(in*BlockSize), "w", bytesize.ByteSize(out*BlockSize))
}
