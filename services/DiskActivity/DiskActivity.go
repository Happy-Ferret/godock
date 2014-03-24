/* Disk activity monitoring applet for the Cairo-Dock project.
 */
package DiskActivity

import (
	"github.com/sqp/godock/libs/cdtype"
	"github.com/sqp/godock/libs/dock"     // Connection to cairo-dock.
	"github.com/sqp/godock/libs/packages" // ByteSize.
	"github.com/sqp/godock/libs/sysinfo"
	"github.com/sqp/godock/libs/ternary" // Ternary operators.

	"fmt"
)

//
//------------------------------------------------------------------[ APPLET ]--

// Applet data and controlers.
//
type Applet struct {
	*dock.CDApplet
	conf    *appletConf
	service *sysinfo.IOActivity
}

// Create a new applet instance.
//
func NewApplet() dock.AppletInstance {
	app := &Applet{CDApplet: dock.NewCDApplet()} // Icon controler and interface to cairo-dock.

	app.service = sysinfo.NewIOActivity(app)
	app.service.FormatIcon = formatIcon
	app.service.FormatLabel = formatLabel
	app.service.GetData = sysinfo.GetDiskActivity

	app.AddPoller(app.service.Check)

	return app
}

// Load user configuration if needed and initialise applet.
//
func (app *Applet) Init(loadConf bool) {
	app.LoadConfig(loadConf, &app.conf) // Load config will crash if fail. Expected.

	// Settings for poller and IOActivity (force renderer reset in case of reload).
	app.conf.UpdateDelay = dock.PollerInterval(app.conf.UpdateDelay, defaultUpdateDelay)
	app.service.Settings(uint64(app.conf.UpdateDelay), cdtype.InfoPosition(app.conf.DisplayText), app.conf.DisplayValues, app.conf.GraphType, app.conf.GaugeName, app.conf.Disks...)

	// Set defaults to dock icon: display and controls.
	app.SetDefaults(dock.Defaults{
		Label:          ternary.String(app.conf.Name != "", app.conf.Name, app.AppletName),
		PollerInterval: app.conf.UpdateDelay,
		Commands: dock.Commands{
			"left":   dock.NewCommandStd(app.conf.LeftAction, app.conf.LeftCommand, app.conf.LeftClass),
			"middle": dock.NewCommandStd(app.conf.MiddleAction, app.conf.MiddleCommand)},
		Debug: app.conf.Debug})
}

//
//------------------------------------------------------------------[ EVENTS ]--

// Define applet events callbacks.
//
func (app *Applet) DefineEvents() {

	// Left and middle clicks: launch configured command.
	app.Events.OnClick = app.LaunchFunc("left")
	app.Events.OnMiddleClick = app.LaunchFunc("middle")

	app.Events.OnBuildMenu = func() {
		menu := []string{}
		if app.conf.LeftAction > 0 {
			menu = append(menu, "Action left click")
		}
		if app.conf.MiddleAction > 0 {
			menu = append(menu, "Action middle click")
		}
		app.PopulateMenu(menu...)
	}

	app.Events.OnMenuSelect = func(i int32) {
		list := []string{"left", "middle"}
		app.LaunchCommand(list[i])
	}
}

//
//-----------------------------------------------------------------[ DISPLAY ]--

// Quick-info display callback. One line for each value. Zero are replaced by empty string.
//
func formatIcon(dev string, in, out uint64) string {
	return sysinfo.FormatRate(in*BlockSize) + "\n" + sysinfo.FormatRate(out*BlockSize)
}

// Label display callback. One line for each device. Format="eth0: ↓ 42 / ↑ 128".
//
func formatLabel(dev string, in, out uint64) string {
	return fmt.Sprintf("%s: %s %s / %s %s", dev, "r", packages.ByteSize(in*BlockSize), "w", packages.ByteSize(out*BlockSize))
}
