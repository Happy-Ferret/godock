// +build dock

package main

import (
	"github.com/pkg/profile" // Pprof file output.

	"github.com/sqp/godock/libs/dock"          // Dock main.
	"github.com/sqp/godock/libs/dock/maindock" // Dock settings.
	"github.com/sqp/godock/libs/gldi"          // Gldi access.
	"github.com/sqp/godock/libs/gldi/globals"  // Global variables.
	"github.com/sqp/godock/libs/net/websrv"    // Web server.
	"github.com/sqp/godock/libs/text/versions" // Print API version.

	"fmt"
	"strings"
)

func init() {
	cmdDefault = &Command{
		Run:       runDock,
		UsageLine: "dock",
		Short:     "cdc starts a custom version of cairo-dock with a new config GUI.",
		Long: `Without command, the dock will be started with those arguments:

Display Backend:
  -c          Use Cairo backend.
  -o          Use OpenGL backend.
  -O          Use OpenGL backend with indirect rendering.
              There are very few case where this option should be used.
  -A          Ask again on startup which backend to use.
  -W          Work around some bugs in Metacity Window-Manager
              (invisible dialogs or sub-docks)

Desktop:
  -k          Lock the dock so that any modification is impossible for users.
  -e env      Force the dock to consider this environnement - use it with care.
  -a          Keep the dock above other windows.
  -s          Don't make the dock appear on all desktops.

Paths override:
  -d path     Use a custom config directory. Default: ~/.config/cairo-dock
  -M path     Ask the dock to load additionnal modules from this directory.
              (though it is unsafe for your dock to load unnofficial modules).
  -S url      Address of a server with additional themes (overrides default).

Web service:
  -host       Adress to listen for the web server (def=localhost).
  -port       Port for the web server (def=15610).

Debug:
  -w time     Wait for N seconds before starting; this is useful if you notice
              some problems when the dock starts with the session.
  -x appname  Exclude applets from activating (still loaded). Separator=;
  -f          Safe mode: don't load any plug-ins.
  -l level    Log verbosity: debug,message,warning,critical,error (def=warning).
  -F          Force to display some output messages with colors.
  -D          Debug mode for the go part of the code (including applets).
  -N          Don't start the internal Dbus service (used by remote command).
  -mon        start web monitor:   http://localhost:15610/debug/charts
                                   http://localhost:15610/debug/pprof
  -pf         pprof file output:   go tool pprof $(which cdc) /pathToFile

Versions:
  -v          Print gldi version.
  -vv         Print all versions.

This version still lacks some options and may not be considered usable for
everybody at the moment. But it also needs to be tested now.
`,
	}

	usageHeader = cmdDefault.Short
	usageFlags = &cmdDefault.Long

	// Dock flags are declared at init.
	var (
		// Display

		userForceCairo     = cmdDefault.Flag.Bool("c", false, "")
		userForceOpenGL    = cmdDefault.Flag.Bool("o", false, "")
		userIndirectOpenGL = cmdDefault.Flag.Bool("O", false, "")
		userAskBackend     = cmdDefault.Flag.Bool("A", false, "")
		userEnv            = cmdDefault.Flag.String("e", "", "")
		userDir            = cmdDefault.Flag.String("d", "", "")
		userThemeServer    = cmdDefault.Flag.String("S", "", "")

		// Maintenance

		userExclude            = cmdDefault.Flag.String("x", "", "")
		userSafeMode           = cmdDefault.Flag.Bool("f", false, "")
		userMetacityWorkaround = cmdDefault.Flag.Bool("W", false, "")
		userVerbosity          = cmdDefault.Flag.String("l", "", "")
		userForceColor         = cmdDefault.Flag.Bool("F", false, "")
		userLocked             = cmdDefault.Flag.Bool("k", false, "")
		userKeepAbove          = cmdDefault.Flag.Bool("a", false, "")
		userNoSticky           = cmdDefault.Flag.Bool("s", false, "")
		userModulesDir         = cmdDefault.Flag.String("M", "", "")

		// New dock settings.

		newWebHost     = cmdDefault.Flag.String("host", websrv.DefaultHost, "")
		newWebPort     = cmdDefault.Flag.Int("port", websrv.DefaultPort, "")
		newWebMonitor  = cmdDefault.Flag.Bool("mon", false, "")
		newDisableDBus = cmdDefault.Flag.Bool("N", false, "")
		newDebug       = cmdDefault.Flag.Bool("D", false, "")
	)

	pprofFile = cmdDefault.Flag.Bool("pf", false, "")

	// Local flags. Common with remote.

	userDelay = cmdDefault.Flag.Int("w", 0, "")

	// Local flags. (static as they do not even start a dock).

	showVersionGldi = cmdDefault.Flag.Bool("v", false, "")
	showVersionAll = cmdDefault.Flag.Bool("vv", false, "")

	// And the returned callback only get the settings once filled.

	dockSettings = func() maindock.DockSettings {
		setPathAbsolute(userDir)

		return maindock.DockSettings{
			ForceCairo:     *userForceCairo,
			ForceOpenGL:    *userForceOpenGL,
			IndirectOpenGL: *userIndirectOpenGL,
			AskBackend:     *userAskBackend,
			Env:            *userEnv,

			UserDefinedDataDir: *userDir,
			ThemeServer:        *userThemeServer,

			Delay:              *userDelay,
			Exclude:            strings.Split(*userExclude, ";"),
			SafeMode:           *userSafeMode,
			MetacityWorkaround: *userMetacityWorkaround,
			Verbosity:          *userVerbosity,
			ForceColor:         *userForceColor,
			Locked:             *userLocked,
			KeepAbove:          *userKeepAbove,
			NoSticky:           *userNoSticky,
			ModulesDir:         *userModulesDir,

			WebHost:     *newWebHost,
			WebPort:     *newWebPort,
			WebMonitor:  *newWebMonitor,
			DisableDBus: *newDisableDBus,
			Debug:       *newDebug,
		}
	}
}

// 	{"maintenance", 'm', G_OPTION_FLAG_IN_MAIN, G_OPTION_ARG_NONE,
// 		&bMaintenance,
// 		_("Allow to edit the config before the dock is started and show the config panel on start."), NULL},
// 	{"exclude", 'x', G_OPTION_FLAG_IN_MAIN, G_OPTION_ARG_STRING,
// 		&cExcludeModule,
// 		_("Exclude a given plug-in from activating (it is still loaded though)."), NULL},

// 	{"testing", 'T', G_OPTION_FLAG_IN_MAIN, G_OPTION_ARG_NONE,
// 		&bTesting,
// 		_("For debugging purpose only. The crash manager will not be started to hunt down the bugs."), NULL},
// 	{"easter-eggs", 'E', G_OPTION_FLAG_IN_MAIN, G_OPTION_ARG_NONE,
// 		&g_bEasterEggs,
// 		_("For debugging purpose only. Some hidden and still unstable options will be activated."), NULL},

var (
	// dockSettings returns maindock settings parsed from the command line.
	dockSettings func() maindock.DockSettings

	// Local flags.

	pprofFile       *bool
	userDelay       *int
	showVersionGldi *bool
	showVersionAll  *bool
)

// runDock starts dock routines and locks the main thread with gtk.
//
func runDock(cmd *Command, args []string) {
	if *pprofFile {
		defer profile.Start().Stop()
	}

	switch {
	case *showVersionGldi:
		fmt.Println(globals.Version()) // -v option only prints gldi version.

	case *showVersionAll:
		versions.Print() // -vv option only prints all versions.

	case dock.Run(logger, dockSettings): // Start dock success => lock gtk.
		dockSettings = nil // free
		gldi.LockGTK()
		maindock.Clean() // may be better with defer, but cause confused panic messages.
	}
}
