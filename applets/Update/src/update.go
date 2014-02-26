/* Update is an applet for Cairo-Dock to check for its new versions and do update.

Install go and get go environment: you need a valid $GOPATH var and directory.

Download, build and install to your Cairo-Dock external applets dir:
  go get github.com/sqp/godock/applets/Update  # download applet and dependencies.

  cd $GOPATH/src/github.com/sqp/godock/applets/Update
  make build  # compile the applet.
  make link   # link the applet to your external applet directory.



TODO: Version checking:
  get a better bzr result than simple revno if the user is on a different branch
   with an other stack of patches. (need to get the split version to know the real number of missing patches)



Icons used:: some icons from the Oxygen pack:
  http://www.iconarchive.com/show/oxygen-icons-by-oxygen-icons.org.1.html


Copyright : (C) 2012-2014 by SQP
E-mail : sqp@glx-dock.org

*/
package main

import (
	"github.com/sqp/godock/libs/cdtype"
	"github.com/sqp/godock/libs/dock" // Connection to cairo-dock.
	"github.com/sqp/godock/libs/log"  // Display info in terminal.
	// "github.com/sqp/godock/libs/poller" // Polling timing handler.

	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

//---------------------------------------------------------------[ MAIN CALL ]--

// Program launched. Create and activate applet.
//
func main() {
	dock.StartApplet(NewApplet())
}

//------------------------------------------------------------------[ APPLET ]--

type AppletUpdate struct {
	*dock.CDApplet             // Dock interface.
	conf           *updateConf // applet user configuration.

	version *Versions   // applet data.
	target  BuildTarget // build from sources interface.

	targetId int // position of current target in BuildTargets list.
	err      error
}

// Create an instance of applet Update. Used to play with cairo-dock sources:
// download/update, compile, restart dock... Usefull for developers, testers
// and users who want to stay up to date, or maybe on a distro without packages.
//
func NewApplet() *AppletUpdate {
	app := &AppletUpdate{
		CDApplet: dock.Applet(),
	}
	app.defineActions()

	// Action indicators: display emblem while busy..
	// onStarted := func() { app.SetEmblem(app.FileLocation("img", app.conf.BuildEmblemWork), EmblemAction) }
	// onFinished := func() { app.SetEmblem("none", EmblemAction) }
	// app.Actions.SetActionIndicators(onStarted, onFinished)

	// Create a cairo-dock sources version checker.
	app.version = &Versions{
		callResult: func(new int, e error) { app.onGotVersions(new, e) },
		newCommits: -1,
	}

	// The poller will check for new versions on a timer.
	poller := app.AddPoller(func() { go app.version.Check() })

	// Set "working" emblem during version check. It should be removed or changed by the check.
	poller.SetPreCheck(func() { app.SetEmblem(app.FileLocation("img", app.conf.VersionEmblemWork), EmblemVersion) })

	return app
}

// Initialise applet with user configuration.
//
func (app *AppletUpdate) Init(loadConf bool) {

	if loadConf { // Try to load config. Exit if not found.
		app.conf = &updateConf{}
		log.Fatal(app.LoadConfig(&app.conf), "config")
	}

	// Icon default settings.
	app.SetDefaults(cdtype.Defaults{
		Icon:           app.conf.Icon,
		Shortkeys:      []string{app.conf.ShortkeyOneKey, app.conf.ShortkeyTwoKey},
		Templates:      []string{app.conf.VersionDialogTemplate},
		PollerInterval: dock.PollerInterval(app.conf.VersionPollingTimer*60, defaultVersionPollingTimer*60),
		Commands: cdtype.Commands{
			"showDiff": cdtype.NewCommand(app.conf.DiffMonitored, app.conf.DiffCommand)},
		Debug: app.conf.Debug})

	// Branches for versions checking.
	app.version.sources = []*Branch{
		NewBranch(app.conf.BranchCore, path.Join(app.conf.SourceDir, app.conf.DirCore)),
		NewBranch(app.conf.BranchApplets, path.Join(app.conf.SourceDir, app.conf.DirApplets))}

	// Build targets. Allow actions on sources and displays emblem on top left for togglable target.
	app.setBuildTarget()

	// Build globals.
	LocationLaunchpad = app.conf.LocationLaunchpad
	cmdSudo = app.conf.CommandSudo
}

//------------------------------------------------------------------[ EVENTS ]--

// Define applet events callbacks.
//
func (app *AppletUpdate) DefineEvents() {

	// Left click: launch configured action for current user mode.
	//
	app.Events.OnClick = func() {
		if app.conf.UserMode {
			log.Info("k", app.conf.DevClickLeft, app.Actions.Id(app.conf.DevClickLeft))
			app.Actions.Launch(app.Actions.Id(app.conf.DevClickLeft))
		} else {
			app.Actions.Launch(app.Actions.Id(app.conf.TesterClickLeft))
		}
	}

	// Middle click: launch configured action for current user mode.
	//
	app.Events.OnMiddleClick = func() {
		if app.conf.UserMode {
			app.Actions.Launch(app.Actions.Id(app.conf.DevClickMiddle))
		} else {
			app.Actions.Launch(app.Actions.Id(app.conf.TesterClickMiddle))
		}
	}

	// Right click menu: show menu for current user mode.
	//
	app.Events.OnBuildMenu = func() {
		if app.conf.UserMode {
			app.BuildMenu(menuDev)
		} else {
			app.BuildMenu(menuTester)
		}
	}

	// Menu entry selected. Launch the expected action.
	//
	app.Events.OnMenuSelect = func(numEntry int32) {
		if app.conf.UserMode {
			app.Actions.Launch(menuDev[numEntry])
		} else {
			app.Actions.Launch(menuTester[numEntry])
		}

	}

	// Scroll event: launch configured action if in dev mode.
	//
	app.Events.OnScroll = func(scrollUp bool) {
		if app.conf.UserMode && app.Actions.Current == 0 { // Wheel action only for dev and if no threaded tasks running.
			id := app.Actions.Id(app.conf.DevMouseWheel)
			if id == ActionCycleTarget { // Cycle depends on wheel direction.
				if scrollUp {
					app.cycleTarget(1)
				} else {
					app.cycleTarget(-1)
				}
			} else { // Other actions are simple toggle.
				app.Actions.Launch(id)
			}
		}
	}

	// Shortkey event: launch configured action.
	//
	app.Events.OnShortkey = func(key string) {
		if key == app.conf.ShortkeyOneKey {
			app.Actions.Launch(app.Actions.Id(app.conf.ShortkeyOneAction))
		}
		if key == app.conf.ShortkeyTwoKey {

			app.Actions.Launch(app.Actions.Id(app.conf.ShortkeyTwoAction))
		}
	}

	// Feature to test: rgrep of the dropped string on the source dir.
	//
	app.Events.OnDropData = func(data string) {
		log.Info("Grep " + data)
		execShow("rgrep", "--color", data, app.ShareDataDir)
	}

}

//----------------------------------------------------------------[ CALLBACK ]--

// Got versions informations, Need to set a new emblem
//
func (app *AppletUpdate) onGotVersions(new int, e error) {
	if new > 0 {
		app.SetEmblem(app.FileLocation("img", app.conf.VersionEmblemNew), EmblemVersion)

		if app.version.newCommits != -1 && new > app.version.newCommits { // Drop first message and only show others if number changed.
			app.actionShowVersions(false)
		}

	} else {
		app.SetEmblem("none", EmblemVersion)
	}
	app.version.newCommits = new
}

//-----------------------------------------------------------------[ ACTIONS ]--

// Define applet actions.
//
func (app *AppletUpdate) defineActions() {
	app.Actions.Max = 1
	app.Actions.Add(
		&dock.Action{
			Id:   ActionNone,
			Menu: 2,
		},
		&dock.Action{
			Id:   ActionShowDiff,
			Name: "Show diff",
			Icon: "gtk-justify-fill",
			Call: func() { app.actionShowDiff() },
		},
		&dock.Action{
			Id:       ActionShowVersions,
			Name:     "Show versions",
			Icon:     "gtk-network", // to change
			Call:     func() { app.actionShowVersions(true) },
			Threaded: true,
		},
		&dock.Action{
			Id:       ActionCheckVersions,
			Name:     "Check versions",
			Icon:     "gtk-network",
			Call:     func() { app.actionCheckVersions() },
			Threaded: true,
		},
		&dock.Action{
			Id:   ActionCycleTarget,
			Name: "Cycle target",
			Icon: "gtk-refresh",
			Call: func() { app.cycleTarget(1) },
		},
		&dock.Action{
			Id:   ActionToggleUserMode,
			Name: "Toggle user mode",
			Menu: 3,
			Call: func() { app.actionToggleUserMode() },
		},

		&dock.Action{
			Id:   ActionToggleReload,
			Name: "Toggle reload action",
			Menu: 3,
			Call: func() { app.actionToggleReload() },
		},
		&dock.Action{
			Id:       ActionBuildTarget,
			Name:     "Build target",
			Icon:     "gtk-media-play",
			Call:     func() { app.actionBuildTarget() },
			Threaded: true,
		},
		//~ action_add(CDCairoBzrAction.GENERATE_REPORT, action_none, "", "gtk-refresh");

		// &dock.Action{
		// 	Id:       ActionBuildAll,
		// 	Name:     "Build All",
		// 	Icon:     "gtk-media-next",
		// 	Call:     func() { app.actionBuildAll() },
		// 	Threaded: true,
		// },
		// &dock.Action{
		// 	Id:       ActionDownloadCore,
		// 	Name:     "Download Core",
		// 	Icon:     "gtk-network",
		// 	Call:     func() { app.actionDownloadCore() },
		// 	Threaded: true,
		// },
		// &dock.Action{
		// 	Id:       ActionDownloadApplets,
		// 	Name:     "Download Plug-Ins",
		// 	Icon:     "gtk-network",
		// 	Call:     func() { app.actionDownloadApplets() },
		// 	Threaded: true,
		// },
		// &dock.Action{
		// 	Id:       ActionDownloadAll,
		// 	Name:     "Download All",
		// 	Icon:     "gtk-network",
		// 	Call:     func() { app.actionDownloadAll() },
		// 	Threaded: true,
		// },
		// &dock.Action{
		// 	Id:       ActionUpdateAll,
		// 	Name:     "Update All",
		// 	Icon:     "gtk-network",
		// 	Call:     func() { app.actionUpdateAll() },
		// 	Threaded: true,
		// },
	)
}

//-----------------------------------------------------------[ BASIC ACTIONS ]--

// Open diff command, or toggle window visibility if application is monitored and opened.
//
func (app *AppletUpdate) actionShowDiff() {
	haveMonitor, hasFocus := app.HaveMonitor()
	switch {
	case app.conf.DiffMonitored && haveMonitor: // Application monitored and open.
		app.ShowAppli(!hasFocus)

	default: // Launch application.
		if _, e := os.Stat(app.target.SourceDir()); e != nil {
			log.Info("Invalid source directory")
		} else {
			execAsync(app.conf.DiffCommand, app.target.SourceDir())
		}
	}
}

// Change target and display the new one.
//
func (app *AppletUpdate) cycleTarget(delta int) {
	app.targetId += delta
	list := app.getBuildTargets()
	if app.targetId >= len(list) {
		app.targetId = 0
	}

	if app.targetId < 0 {
		app.targetId = len(list) - 1
	}

	app.setBuildTarget()
	app.showTarget()
}

func (app *AppletUpdate) actionToggleUserMode() {
	app.conf.UserMode = !app.conf.UserMode
	app.setBuildTarget()
}

func (app *AppletUpdate) actionToggleReload() {
	app.conf.BuildReload = !app.conf.BuildReload
}

//--------------------------------------------------------[ THREADED ACTIONS ]--

// Check new versions now and reset timer.
//
func (app *AppletUpdate) actionCheckVersions() {
	app.Poller().Restart()
}

// To improve : parse http://bazaar.launchpad.net/~cairo-dock-team/cairo-dock-core/cairo-dock/changes/
// and maybe see to use as download tool : http://golang.org/src/cmd/go/vcs.go
//
func (app *AppletUpdate) actionShowVersions(force bool) {
	for _, v := range app.version.Sources() {
		if v.Delta > 0 {
			force = true
		}
	}
	if force {
		text, e := app.ExecuteTemplate(app.conf.VersionDialogTemplate, app.conf.VersionDialogTemplate, app.version.Sources())
		log.Err(e, "template "+app.conf.VersionDialogTemplate)
		// log.Info("Dialog", text)

		dialog := map[string]interface{}{
			"message":     text,
			"use-markup":  true,
			"time-length": int32(app.conf.VersionDialogTimer),
		}

		log.Err(app.PopupDialog(dialog, nil), "popup")

	}
}

// Build current target.
//
func (app *AppletUpdate) actionBuildTarget() {
	app.AddDataRenderer("progressbar", 1, "")
	defer app.AddDataRenderer("progressbar", 0, "")

	// app.Animate("busy", 200)
	if !log.Err(app.target.Build(), "Build") {
		log.Info("Build", app.target.Label())
		app.restartTarget()
	}
}

// func (app *AppletUpdate) actionBuildCore()       {}
// func (app *AppletUpdate) actionBuildApplets()    {}
func (app *AppletUpdate) actionBuildAll()        {}
func (app *AppletUpdate) actionDownloadCore()    {}
func (app *AppletUpdate) actionDownloadApplets() {}
func (app *AppletUpdate) actionDownloadAll()     {}
func (app *AppletUpdate) actionUpdateAll()       {}

//------------------------------------------------------------------[ COMMON ]--

func activityBar(c <-chan time.Time, render func(float64)) {
	var val, step float64
	step = 0.05
	for _ = range c {
		if val+step < 0 || 1 < val+step {
			step = -step
		}
		val += step
		render(val)
	}
}

// Run command with output forwarded to console.
//
func execShow(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	e := cmd.Run()
	//~ if e != nil {
	//~ log.Println(e)
	//~ }
	return e
}

func execSync(command string, args ...string) (string, error) {
	out, e := exec.Command(command, args...).Output()
	//~ if logE(command, "execSync: " + e.Error()) {
	if e != nil {
		args = append([]string{command}, args...)
		//~ log.Err(e, "execSync error launching : " + command + " " + args[0])
		log.Err(e, "execSync: "+strings.Join(args, " "))
		//~ if logE(command, e) {
		//~ log.Debug("execSync error launching : " + command, args)
		//~ return ""
	}
	//~ term.Info("exec", command, args, string(out))
	return string(out), e
}

func execAsync(command string, args ...string) error {
	return log.GetErr(exec.Command(command, args...).Start(), "Execute failed "+command)
	//~ e := exec.Command(command, args...).Start()
	//~ if e != nil {
	//~ log.Println(term.Red("Launch failed error"), e, args)
	//~ }
	//~ return e
}

// Test error: log it.
//
func logE(action string, e error) (wasErr bool) {
	if e != nil {
		wasErr = true
		log.Err(e, "Error")
	}
	return wasErr
}

// Test error: log and quit.
//
func testFatal(e error) {
	if e != nil {
		//~ log.Println(term.Red("Applet load error"), e)
		os.Exit(2)
	}
}

// Get numeric part of a string and convert it to int.
//
// func trimInt(imdb string) (int, error) {
// 	//~ Replace, _ := regexp.CompilePOSIX("^.*([:digit:]*).*$")
// 	Replace, _ := regexp.Compile("[0-9]+")
// 	str := Replace.FindString(imdb)
// 	return strconv.Atoi(str)
// }

func trimInt(str string) (int, error) {
	return strconv.Atoi(strings.Trim(str, "  \n"))
}

// Ternary operator for int. return (test ? a : b)
//
func testInt(test bool, a, b int) int {
	if test {
		return a
	}
	return b
}
