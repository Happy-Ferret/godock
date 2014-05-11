package dock

import (
	"github.com/guelfey/go.dbus" // imported as dbus.

	"github.com/sqp/godock/libs/appdbus" // Connection to cairo-dock.
	"github.com/sqp/godock/libs/cdtype"
	"github.com/sqp/godock/libs/config"
	"github.com/sqp/godock/libs/log" // Display info in terminal.
	"github.com/sqp/godock/libs/poller"

	"bytes"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"
)

// RenderSimple is a small interface to the Dock icon for simple renderers like data pollers.
//
type RenderSimple interface {
	AddDataRenderer(string, int32, string) error
	FileLocation(...string) string
	RenderValues(...float64) error
	SetIcon(string) error
	SetLabel(string) error
	SetQuickInfo(string) error
}

//
//------------------------------------------------------------[ START APPLET ]--

// AppletInstance is the list of methods an applet must implement to use the StartApplet func.
//
type AppletInstance interface {
	// Need to be defined in user applet.
	Init(loadConf bool)

	// DefineEvents() // optional.

	// Defined by CDApplet
	AddPoller(call func()) *poller.Poller
	Poller() *poller.Poller
	SetEventReload(initFunc func(loadConf bool)) // Forward the init callback from interface to the reload event.

	// Defined by CDDbus
	ConnectToBus() (<-chan *dbus.Signal, error)
	ConnectEvents(conn *dbus.Conn) error
	RegisterEvents(interface{}) []error
	SetArgs(args []string)
	OnSignal(*dbus.Signal) (exit bool)
}

type defineEventser interface {
	DefineEvents()
}

// StartApplet will prepare and launch a cairo-dock applet. If you have provided
// events, they will respond when needed, and you have nothing more to worry
// about your applet management. It can handle only one poller for now.
//
// List of the steps, and their effect:
//   * Load applet events definition = DefineEvents().
//   * Connect the applet to cairo-dock with DBus. This also activate events callbacks.
//   * Initialise applet with option load config activated = Init(true).
//   * Start and run the polling loop if needed. This start a instant check, and
//     manage regular and manual timer refresh.
//   * Wait for the dock End signal to close the applet.
//
func StartApplet(app AppletInstance) {
	if app == nil {
		// log.Info("Applet failed to start")
		return
	}

	log.Debug("Applet started")
	defer log.Debug("Applet stopped")

	// Define and connect events to the dock.
	app.SetArgs(os.Args)

	app.SetEventReload(func(loadConf bool) { app.Init(loadConf) })

	if d, ok := app.(defineEventser); ok { // Old events callback method.
		d.DefineEvents()
	}

	dbusEvent, e := app.ConnectToBus()
	log.Fatal(e, "ConnectToBus") // Mandatory.

	app.RegisterEvents(app) // New events callback method.

	// Initialise applet: Load config and apply user settings.
	app.Init(true)

	if poller := app.Poller(); poller != nil {

		restart := make(chan string, 1)
		poller.SetChanRestart(restart, "1") // Restart chan for user events.
		action := true                      // Launch the poller check action directly at start.

		for { // Start main loop and handle events until the End signal is received from the dock.

			if action { // Launch the poller check action.
				go poller.Action()
				action = false
			}

			select { // Wait for events. Until the End signal is received from the dock.

			case s := <-dbusEvent: // Listen to DBus events.
				if app.OnSignal(s) {
					return // Signal was stop_module. That's all folks. We're closing.
				}

			case <-poller.Wait(): // Wait for the end of the timer. Reloop and check.
				action = true

			case <-restart: // Wait for manual restart event. Reloop and check.
				action = true
			}
		}

	} else { // Just handle DBus events until stop_module event.
		for s := range dbusEvent {
			if app.OnSignal(s) {
				return // Signal was stop_module. That's all folks. We're closing.
			}
		}
	}
}

//
//----------------------------------------------------------------[ CDAPPLET ]--

// CDApplet is the base Cairo-Dock applet manager that will handle all your
// communications with the dock and provide some methods commonly needed by
// applets.
//
type CDApplet struct {
	AppletName    string // Applet name as known by the dock. As an external app = dir name.
	ConfFile      string // Config file location.
	ParentAppName string // Application launching the applet.
	ShareDataDir  string // Location of applet data files. As an external applet, it is the same as binary dir.
	RootDataDir   string //

	Templates map[string]*template.Template // Templates for text formating.
	Actions   Actions                       // Actions handler. Where events callbacks must be declared.
	commands  Commands                      // Programs and locations configured by the user, including application monitor.
	poller    *poller.Poller                // Poller loop. Need to provide a way to use more than one.
	Log       cdtype.Logger                 // Applet logger.

	*appdbus.CDDbus // Dbus connector.
}

// NewCDApplet creates a new applet manager with arguments received from command line.
//
func NewCDApplet() *CDApplet {
	cda := &CDApplet{
		Templates: make(map[string]*template.Template),
		Log:       log.NewLog(log.Logs),
	}
	return cda
}

// SetArgs load settings with the list of args received from command launch.
//
func (cda *CDApplet) SetArgs(args []string) {
	name := args[0][2:] // Strip ./ in the beginning.
	// log.SetPrefix(name)
	cda.Log.SetName(name)

	cda.AppletName = name
	cda.ConfFile = args[3]
	cda.RootDataDir = args[4]
	cda.ParentAppName = args[5]

	// TODO: need to find a better way to set the current dir.
	// cda.ShareDataDir = path.Join(args[4], appletsDir, name)
	if len(args) > 7 {
		cda.ShareDataDir = args[7] // dir forwarded from the launcher.
	} else {
		var e error
		cda.ShareDataDir, e = os.Getwd() // standalone applet, using current dir.
		cda.Log.Err(e, "get applet dir")
	}
	cda.CDDbus = appdbus.New(args[2])

	cda.CDDbus.Log = cda.Log
}

// SetEventReload set the default reload event with the applet init callback.
//
func (cda *CDApplet) SetEventReload(appInit func(loadConf bool)) {
	if cda.Events.Reload == nil {
		cda.Events.Reload = func(confChanged bool) {
			cda.Log.Debug("Reload module")
			appInit(confChanged)
			if cda.poller != nil {
				cda.poller.Restart() // send our restart event.
			}
		}
	}
}

//
//----------------------------------------------------------------[ DEFAULTS ]--

// Defaults settings that can be set in one call with something like:
//    app.SetDefaults(dock.Defaults{
//        Label:      "No data",
//        QuickInfo:  "?",
//    })
//
type Defaults struct {
	Icon      string
	Label     string
	QuickInfo string
	Shortkeys []string

	PollerInterval int
	Commands       Commands

	Templates []string
	Debug     bool // Enable debug flood.
}

// SetDefaults set basic defaults icon settings in one call. Empty fields will
// be reset, so this is better used in the Init() call.
//
func (cda *CDApplet) SetDefaults(def Defaults) {
	icon := def.Icon
	if icon == "" {
		icon = cda.FileLocation("icon")
	}
	cda.SetIcon(icon)
	cda.SetQuickInfo(def.QuickInfo)
	cda.SetLabel(def.Label)
	cda.BindShortkey(def.Shortkeys...)

	cda.commands = def.Commands
	cda.ControlAppli(cda.commands.FindMonitor())

	if poller := cda.Poller(); poller != nil {
		poller.SetInterval(def.PollerInterval)
	}

	cda.LoadTemplate(def.Templates...)
	// log.SetDebug(def.Debug)

	cda.Log.SetDebug(def.Debug)
}

//
//---------------------------------------------------------------[ TEMPLATES ]--

// LoadTemplate load the provided list of template files. If error, it will just be be logged, so you must check
// that the template is valid. Map entry will still be created, just check if it
// isn't nil. *CDApplet.ExecuteTemplate does it for you.
//
// Templates must be in a subdir called templates in applet dir. If you really
// need a way to change this, ask for a new method.
//
func (cda *CDApplet) LoadTemplate(names ...string) {
	for _, name := range names {
		fileloc := cda.FileLocation("templates", name+".tmpl")
		template, e := template.ParseFiles(fileloc)
		cda.Log.Err(e, "Template")
		cda.Templates[name] = template
	}
}

// ExecuteTemplate will run a pre-loaded template with the given data.
//
func (cda *CDApplet) ExecuteTemplate(file, name string, data interface{}) (string, error) {
	if cda.Templates[file] == nil {
		return "", fmt.Errorf("missing template %s", file)
	}

	buff := bytes.NewBuffer([]byte(""))
	if e := cda.Templates[file].ExecuteTemplate(buff, name, data); cda.Log.Err(e, "FormatDialog") {
		return "", e
	}
	return buff.String(), nil
}

//
//------------------------------------------------------------------[ POLLER ]--

// AddPoller add a poller to handle in the main loop. Only one can be active ATM.
// API will almost guaranteed to change for the sub functions.
//
func (cda *CDApplet) AddPoller(call func()) *poller.Poller {
	cda.poller = poller.New(call)
	return cda.poller
}

// Poller return the applet poller if any.
//
func (cda *CDApplet) Poller() *poller.Poller {
	return cda.poller
}

//
//----------------------------------------------------------------[ COMMANDS ]--

// HaveMonitor gives informations about the state of the monitored application.
// Those are usefull is this option is enabled. A monitored application, if
// opened, is supposed to have its visibility state toggled by the user event.
//
//  haveApp: true if the monitored application is opened. (Xid > 0)
//  HaveFocus: true if the monitored application is the one with the focus.
//
func (cda *CDApplet) HaveMonitor() (haveApp bool, haveFocus bool) {
	Xid, e := cda.Get("Xid")
	cda.Log.Err(e, "Xid")

	if id, ok := Xid.(uint64); ok {
		haveApp = id > 0
	}
	HasFocus, _ := cda.Get("has_focus")
	return haveApp, HasFocus.(bool)
}

// LaunchCommand executes one of the configured command by its reference.
//
func (cda *CDApplet) LaunchCommand(name string) {
	if cmd, ok := cda.commands[name]; ok {
		if cmd.Monitored {
			haveMonitor, hasFocus := cda.HaveMonitor()
			if haveMonitor { // Application monitored and opened.
				cda.ShowAppli(!hasFocus)
				return
			}
		}
		splitted := strings.Split(cmd.Name, " ")
		if cmd.UseOpen {
			cda.Log.ExecAsync("xdg-open", splitted...)
		} else {
			cda.Log.ExecAsync(splitted[0], splitted[1:]...)
		}
	}
}

// LaunchFunc returns a callback to a configured command to bind with event
// OnClick or OnMiddleClick.
//
func (cda *CDApplet) LaunchFunc(name string) func() {
	return func() { cda.LaunchCommand(name) }
}

// Commands handles a list of Command.
//
type Commands map[string]*Command

// FindMonitor return the configured window class for the command.
//
func (commands Commands) FindMonitor() string {
	for _, cmd := range commands {
		if cmd.Monitored {
			if cmd.Class != "" { // Class provided, use it.
				return cmd.Class
			}
			return cmd.Name // Else use program name.
		}
	}
	return "none" // None found, reset it.
}

// Command is the description of a standard command launcher.
//
type Command struct {
	Name      string // Command or location to open.
	UseOpen   bool   // If true, open with xdg-open.
	Monitored bool   // If true, the window will be monitored by the dock. (don't work wit UseOpen)
	Class     string // Window class if needed.
}

// NewCommand creates a standard command launcher.
//
func NewCommand(monitored bool, name string, class ...string) *Command {
	cmd := &Command{
		Monitored: monitored,
		Name:      name,
	}
	if len(class) > 0 {
		cmd.Class = class[0]
	}
	return cmd
}

// NewCommandStd creates a command launcher from configuration options.
//
func NewCommandStd(action int, name string, class ...string) *Command {
	cmd := NewCommand(action == 3, name, class...)
	cmd.UseOpen = (action == 1)
	return cmd
}

//
//-----------------------------------------------------------------[ HELPERS ]--

// LoadConfig will try to create and fill the given config struct with data from
// the configuration file. Log error and crash if something went wrong. Does
// nothing if loadConf is false.
//
func (cda *CDApplet) LoadConfig(loadConf bool, v interface{}) {
	if loadConf { // Try to load config. Exit if not found.
		log.Fatal(config.Load(cda.ConfFile, v, config.GetBoth), "config")
	}
}

// FileLocation return the full path to a file in the applet data dir.
//
func (cda *CDApplet) FileLocation(filename ...string) string {
	args := append([]string{cda.ShareDataDir}, filename...)
	return path.Join(args...)
}

// SetDebug set the state of the debug reporting flood.
//
func (cda *CDApplet) SetDebug(debug bool) {
	cda.Log.SetDebug(debug)
}

// PollerInterval sets the poller check interval.
//
func PollerInterval(val ...int) int {
	for _, d := range val {
		if d > 0 {
			return d
		}
	}
	return 3600 * 24 // Failed to provide a valid value. Set check interval to one day.
}

//
//-------------------------------------------------------------[ MENU SIMPLE ]--

// Menu is a really simple menu to store callbacks at creation to be sure the
// answer match the user request.
// It's gonna evolve when we'll have access to the better menu build method, but
// its goal will stay the same.
//
type Menu struct {
	Actions []func() // Menu callbacks are saved to be sure we launch the good action (options can change).
	Names   []string
}

// Append an item to the menu with its callback.
//
func (menu *Menu) Append(name string, call func()) {
	menu.Names = append(menu.Names, name)
	menu.Actions = append(menu.Actions, call)
}

// Launch calls the action referenced by its id.
func (menu *Menu) Launch(id int32) {
	if int(id) < len(menu.Actions) {
		menu.Actions[id]()
	}
}

// Clear resets the menu items list.
//
func (menu *Menu) Clear() {
	menu.Actions = nil
	menu.Names = nil
}
