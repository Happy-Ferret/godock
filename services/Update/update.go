/*
Package Update is an applet for Cairo-Dock to build and update the dock and applets.

Play with cairo-dock sources:
download/update, compile, restart dock... Usefull for developers, testers and
users who want to stay up to date, or maybe on a distro without packages.
*/
package Update

/*
Copyright : (C) 2012-2015 by SQP
E-mail    : sqp@glx-dock.org

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 3
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.
http://www.gnu.org/licenses/licenses.html#GPL
*/

import (
	"github.com/sqp/godock/libs/cdapplet"       // Applet base.
	"github.com/sqp/godock/libs/cdtype"         // Applet types.
	"github.com/sqp/godock/libs/clipboard"      // Get clipboard content.
	"github.com/sqp/godock/libs/packages/build" // Sources builder.
	"github.com/sqp/godock/libs/text/linesplit" // Parse command output.

	"errors"
	"fmt"
	"os"
	"path"
	// "path/filepath"
	"strconv"
	"strings"
)

//------------------------------------------------------------------[ APPLET ]--

// Applet data and controlers.
//
type Applet struct {
	cdtype.AppBase // Applet base and dock connection.

	conf    *updateConf   // applet user configuration.
	version *Versions     // applet data.
	target  build.Builder // build from sources interface.

	targetID int // position of current target in BuildTargets list.
	err      error
}

// NewApplet create an new Update applet instance.
//
func NewApplet() cdtype.AppInstance {
	app := &Applet{AppBase: cdapplet.New()}
	app.defineActions()

	// Create a cairo-dock sources version checker.
	app.version = &Versions{
		callResult: app.onGotVersions,
		newCommits: -1,
	}

	// The poller will check for new versions on a timer.
	poller := app.Poller().Add(app.version.Check)

	// Set "working" emblem during version check. It should be removed or changed by the check.
	poller.SetPreCheck(func() { app.SetEmblem(app.FileLocation("img", app.conf.VersionEmblemWork), EmblemVersion) })

	return app
}

// Init load user configuration if needed and initialise applet.
//
func (app *Applet) Init(loadConf bool) {
	app.LoadConfig(loadConf, &app.conf) // Load config will crash if fail. Expected.

	// Icon default settings.
	app.SetDefaults(cdtype.Defaults{
		Icon:           app.conf.Icon,
		Label:          app.conf.Name,
		Templates:      []string{app.conf.VersionDialogTemplate},
		PollerInterval: cdtype.PollerInterval(app.conf.VersionPollingTimer*60, defaultVersionPollingTimer*60),
		Commands: cdtype.Commands{
			cmdShowDiff: cdtype.NewCommand(app.conf.DiffMonitored, app.conf.DiffCommand)},
		Shortkeys: []cdtype.Shortkey{
			{"Actions", "ShortkeyOneKey", "Action one", app.conf.ShortkeyOneKey},
			{"Actions", "ShortkeyTwoKey", "Action two", app.conf.ShortkeyTwoKey}},
		Debug: app.conf.Debug})

	if app.conf.VersionPollingEnabled {
		app.Poller().Start()
	} else {
		app.Poller().Stop()
	}

	// Branches for versions checking.
	app.version.sources = []*Repo{
		NewRepo(app.Log(), app.conf.BranchCore, path.Join(app.conf.SourceDir, app.conf.DirCore)),
		NewRepo(app.Log(), app.conf.BranchApplets, path.Join(app.conf.SourceDir, app.conf.DirApplets)),
	}
	if app.conf.SourceExtra != "" {
		sources := strings.Split(app.conf.SourceExtra, "\\n")
		for _, src := range sources {
			app.Log().Info(path.Base(src), src)
			app.version.sources = append(app.version.sources, NewRepo(app.Log(), path.Base(src), src))
		}
	}

	// Build targets. Allow actions on sources and displays emblem on top left for togglable target.
	app.setBuildTarget()

	// Build globals.
	build.CmdSudo = app.conf.CommandSudo
	build.IconMissing = app.FileLocation("img", app.conf.IconMissing)

	// Set booleans references for menu checkboxes.
	app.Action().SetBool(ActionToggleUserMode, &app.conf.UserMode)
	app.Action().SetBool(ActionToggleReload, &app.conf.BuildReload)
	app.Action().SetBool(ActionToggleDiffStash, &app.conf.DiffStash)
}

//------------------------------------------------------------------[ EVENTS ]--

// DefineEvents set applet events callbacks.
//
func (app *Applet) DefineEvents(events *cdtype.Events) {

	// Left click: launch configured action for current user mode.
	//
	events.OnClick = func(int) {
		if app.conf.UserMode {
			app.Action().Launch(app.Action().ID(app.conf.DevClickLeft))
		} else {
			app.Action().Launch(app.Action().ID(app.conf.TesterClickLeft))
		}
	}

	// Middle click: launch configured action for current user mode.
	//
	events.OnMiddleClick = func() {
		if app.conf.UserMode {
			app.Action().Launch(app.Action().ID(app.conf.DevClickMiddle))
		} else {
			app.Action().Launch(app.Action().ID(app.conf.TesterClickMiddle))
		}
	}

	// Right click menu: show menu for current user mode.
	//
	events.OnBuildMenu = func(menu cdtype.Menuer) {
		if app.conf.UserMode {
			dev := menuDev
			if len(app.version.sources) > 2 {
				dev = append(dev, ActionDownloadOthers)
			}
			app.Action().BuildMenu(menu, dev)
		} else {
			app.Action().BuildMenu(menu, menuTester)
		}
	}

	// Scroll event: launch configured action if in dev mode.
	//
	events.OnScroll = func(scrollUp bool) {
		// app.Log().Info("scroll", app.conf.UserMode, app.ActionCount(), app.ActionID(app.conf.DevMouseWheel))
		if !app.conf.UserMode || app.Action().Count() > 0 { // Wheel action only for dev and if no threaded tasks running.
			return
		}
		id := app.Action().ID(app.conf.DevMouseWheel)
		if id == ActionCycleTarget { // Cycle depends on wheel direction.
			if scrollUp {
				app.actionCycleTarget(1)
			} else {
				app.actionCycleTarget(-1)
			}
		} else { // Other actions are simple toggle.
			app.Action().Launch(id)
		}
	}

	// Shortkey event: launch configured action.
	//
	events.OnShortkey = func(key string) {
		if key == app.conf.ShortkeyOneKey {
			app.Action().Launch(app.Action().ID(app.conf.ShortkeyOneAction))
		}
		if key == app.conf.ShortkeyTwoKey {
			app.Action().Launch(app.Action().ID(app.conf.ShortkeyTwoAction))
		}
	}

	// Feature to test: rgrep of the dropped string on the source dir.
	//
	events.OnDropData = app.actionGrepTarget
}

//----------------------------------------------------------------[ CALLBACK ]--

// onGotVersions is triggered after a version check, Need to set a new emblem.
//
func (app *Applet) onGotVersions(new int, e error) {
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
// Actions order in this list must match the order of defined actions numbers.
//
func (app *Applet) defineActions() {
	app.Action().SetMax(1)
	app.Action().Add(
		&cdtype.Action{
			ID:   ActionNone,
			Menu: cdtype.MenuSeparator,
		},
		&cdtype.Action{
			ID:   ActionShowDiff,
			Name: "Show diff",
			Icon: "format-justify-fill",
			Call: app.actionShowDiff,
		},
		&cdtype.Action{
			ID:       ActionShowVersions,
			Name:     "Show versions",
			Icon:     "network-workgroup", // to change
			Call:     func() { app.actionShowVersions(true) },
			Threaded: true,
		},
		&cdtype.Action{
			ID:       ActionCheckVersions,
			Name:     "Check versions",
			Icon:     "network-workgroup",
			Call:     app.actionCheckVersions,
			Threaded: true,
		},
		&cdtype.Action{
			ID:       ActionGrepTarget,
			Name:     "Grep target",
			Icon:     "view-refresh",
			Call:     app.actionGrepTargetClip,
			Threaded: false,
		},
		&cdtype.Action{
			ID:       ActionCycleTarget,
			Name:     "Cycle target",
			Icon:     "view-refresh",
			Call:     func() { app.actionCycleTarget(1) },
			Threaded: true,
		},
		&cdtype.Action{
			ID:   ActionToggleUserMode,
			Name: "Dev mode",
			Menu: cdtype.MenuCheckBox,
			Call: app.actionToggleUserMode,
		},

		&cdtype.Action{
			ID:   ActionToggleReload,
			Name: "Reload target",
			Menu: cdtype.MenuCheckBox,
			Call: app.actionToggleReload,
		},
		&cdtype.Action{
			ID:   ActionToggleDiffStash,
			Name: "Diff vs stash",
			Menu: cdtype.MenuCheckBox,
			// Call: app.actionToggleReload,
		},
		&cdtype.Action{
			ID:       ActionBuildTarget,
			Name:     "Build target",
			Icon:     "media-playback-start",
			Call:     app.actionBuildTarget,
			Threaded: true,
		},
		//~ action_add(CDCairoBzrAction.GENERATE_REPORT, action_none, "", "view-refresh");

		// &cdtype.Action{
		// 	ID:       ActionBuildAll,
		// 	Name:     "Build All",
		// 	Icon:     "media-skip-forward",
		// 	Call:     func() { app.actionBuildAll() },
		// 	Threaded: true,
		// },
		// &cdtype.Action{
		// 	ID:       ActionDownloadCore,
		// 	Name:     "Download Core",
		// 	Icon:     "network-workgroup",
		// 	Call:     func() { app.actionDownloadCore() },
		// 	Threaded: true,
		// },
		// &cdtype.Action{
		// 	ID:       ActionDownloadApplets,
		// 	Name:     "Download Plug-Ins",
		// 	Icon:     "network-workgroup",
		// 	Call:     func() { app.actionDownloadApplets() },
		// 	Threaded: true,
		// },
		// &cdtype.Action{
		// 	ID:       ActionDownloadAll,
		// 	Name:     "Download All",
		// 	Icon:     "network-workgroup",
		// 	Call:     func() { app.actionDownloadAll() },
		// 	Threaded: true,
		// },
		&cdtype.Action{
			ID:       ActionUpdateAll,
			Name:     "Update All",
			Icon:     "network-workgroup",
			Call:     func() { go app.actionUpdateAll() }, // Threaded as it blocks everything in dock mode.
			Threaded: true,
		},
		&cdtype.Action{
			ID:       ActionDownloadOthers,
			Name:     "Download others",
			Icon:     "network-workgroup",
			Call:     app.actionUpdateOthers,
			Threaded: true,
		},
	)
}

//-----------------------------------------------------------[ BASIC ACTIONS ]--

// Open diff command, or toggle window visibility if application is monitored and opened.
//
func (app *Applet) actionShowDiff() {
	var e error
	switch {
	case app.conf.DiffMonitored && app.Window().IsOpened(): // Application monitored and open.
		e = app.Window().ToggleVisibility()

	default: // Launch application.
		dir := app.target.SourceDir()
		if _, e = os.Stat(dir); e != nil {
			e = errors.New("invalid source directory: " + dir)
		} else {
			if app.conf.DiffStash {
				e = app.Log().ExecAsync("git", "-C", dir, "difftool", "-d")
			} else {
				e = app.Log().ExecAsync(app.conf.DiffCommand, dir)
			}
		}
	}
	app.Log().Err(e, "show diff")
}

// actionGrepTarget searches the directory for the given string.
//
func (app *Applet) actionGrepTarget(search string) {
	if len(search) < 2 { // security, need to confirm or improve.
		app.Log().NewErr("grep", "search query too short, need at least 2 chars")
		return
	}

	// Escape ." chars (dot and quotes).
	query := strings.Replace(search, "\"", "\\\"", -1)
	query = strings.Replace(query, ".", "\\.", -1)

	// Prepare command.
	out := ""
	count := 0
	cmd := app.Log().ExecCmd("grep", append(grepCmdArgs, query)...) // get the command with default args.
	cmd.Dir = app.target.SourceDir()                                // set command dir to reduce file path.
	cmd.Stdout = linesplit.NewWriter(func(s string) {               // results display formatter.
		count++
		sp := strings.SplitN(s, ":", 2)
		if len(sp) == 2 {
			out += grepFileFormatter(sp[0]) + ":\t" // start line with percent and a tab.
			colored := strings.Replace(sp[1], search, grepQueryFormatter(search), -1)
			out += strings.TrimLeft(colored, " \t") + "\n" // remove space and tab.

		} else {
			out += s + "\n"
		}
	})

	// Launch command.
	e := cmd.Run()
	app.Log().Err(e, "Grep target")

	// Print title and list.
	found := "none found"
	if count > 0 {
		found = fmt.Sprintf("count %d", count)
	}
	fmt.Printf(grepTitlePattern, grepTitleFormatter(search), found)
	println(out)
}

// actionGrepTargetClip searches the directory using the clipboard content as
// search pattern.
//
func (app *Applet) actionGrepTargetClip() {
	search, e := clipboard.Read()
	if !app.Log().Err(e, "clipboard.Read") {
		app.actionGrepTarget(search)
	}
}

// actionCycleTarget changes the target and display the new one.
//
func (app *Applet) actionCycleTarget(delta int) {
	app.targetID += delta
	switch {
	case app.targetID >= len(app.conf.BuildTargets):
		app.targetID = 0

	case app.targetID < 0:
		app.targetID = len(app.conf.BuildTargets) - 1
	}

	app.setBuildTarget()
	app.showTarget()
}

func (app *Applet) actionToggleUserMode() {
	app.conf.UserMode = !app.conf.UserMode
	app.setBuildTarget()
}

func (app *Applet) actionToggleReload() {
	app.conf.BuildReload = !app.conf.BuildReload
}

//--------------------------------------------------------[ THREADED ACTIONS ]--

// Check new versions now and reset timer.
//
func (app *Applet) actionCheckVersions() {
	app.Poller().Restart()
}

// Show new versions popup.
//
func (app *Applet) actionShowVersions(force bool) {
	for _, v := range app.version.Sources() {
		if v.Delta > 0 {
			force = true
		}
	}
	if force {
		text, e := app.Template().Execute(app.conf.VersionDialogTemplate, app.conf.VersionDialogTemplate, app.version.Sources())
		if app.Log().Err(e, "template "+app.conf.VersionDialogTemplate) {
			return
		}

		app.PopupDialog(cdtype.DialogData{
			Message:    text,
			TimeLength: app.conf.VersionDialogTimer,
			UseMarkup:  true,
			// Buttons:    "document-open;cancel",
		})
		app.Log().Err(e, "popup")
	}
}

// Build current target.
//
func (app *Applet) actionBuildTarget() {
	app.DataRenderer().Progress(1)
	defer app.DataRenderer().Remove()

	// app.Animate("busy", 200)
	if !app.Log().Err(app.target.Build(), "Build") {
		app.Log().Info("Build", app.target.Label())
		app.restartTarget()
	}
}

// func (app *Applet) actionBuildCore()       {}
// func (app *Applet) actionBuildApplets()    {}
func (app *Applet) actionBuildAll()        {}
func (app *Applet) actionDownloadCore()    {}
func (app *Applet) actionDownloadApplets() {}
func (app *Applet) actionDownloadAll()     {}

// actionUpdateAll download and rebuild the dock core and all applets.
//
func (app *Applet) actionUpdateAll() {
	app.DataRenderer().Progress(1)
	defer app.DataRenderer().Remove()

	// Core.
	_, _, e := app.version.sources[0].update()
	if app.Log().Err(e, "update core") {
		return
	}

	app.Log().Info("updating core")
	core := &build.BuilderCore{}
	core.SetDir(app.conf.SourceDir)
	core.SetProgress(func(f float64) { app.DataRenderer().Render(f) })
	e = core.Build()
	if app.Log().Err(e, "build core") {
		return
	}

	// Plug-ins.
	_, _, e = app.version.sources[1].update()
	if app.Log().Err(e, "update applets") {
		return
	}

	app.Log().Info("updating applets")
	applets := &build.BuilderApplets{}
	applets.SetDir(app.conf.SourceDir)
	applets.SetProgress(func(f float64) { app.DataRenderer().Render(f) })

	applets.MakeFlags = "-Denable-Logout=no" // "-Denable-gmenu=no"

	app.Log().Err(applets.Build(), "build applets")

	app.Poller().Restart()
}

// actionUpdateOthers update extra git sources (hidden option, use key SourceExtra).
//
func (app *Applet) actionUpdateOthers() {
	for _, src := range app.version.sources[2:] {
		_, _, e := src.update()
		app.Log().Err(e, "download", src.Name)
	}
	app.Poller().Restart()
}

//
//------------------------------------------------------------------[ COMMON ]--

// Get numeric part of a string and convert it to int.
//
// func trimInt(imdb string) (int, error) {
// 	//~ Replace, _ := regexp.CompilePOSIX("^.*([:digit:]*).*$")
// 	Replace, _ := regexp.Compile("[0-9]+")
// 	str := Replace.FindString(imdb)
// 	return strconv.Atoi(str)
// }

func trimInt(str string) (int, error) {
	return strconv.Atoi(strings.Trim(str, " \n"))
}
