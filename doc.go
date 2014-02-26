/*
Package godock is a library to build Cairo-Dock applets with the DBus connector.

This API just started to exist, so it will evolve, but not that much. You can
already consider starting an applet using it. All the base actions and callbacks
are provided by Cairo-Dock, and are in stable state, so everything provided will
remain in almost the same format. Only the name, args, of way to access methods
may evolve in the Golang implementation, but only to better suits the needs and
provide a great Cairo-Dock Golang API. Those changes could be made to fix some
issues you may encounter while using it, so feel free to use it and post your
comments on the forum.

Cairo-Dock : http://glx-dock.org/

Usefull links:
	* Main Cairo-Dock Applet API, with everything to build an applet: http://github.com/sqp/godock/libs/dock
	* The DBus connector, the methods to directly talk to the dock: http://github.com/sqp/godock/libs/dbus
	* The default types defined by the dock API. Also has description of events: http://github.com/sqp/godock/libs/cdtype
Could also help applet developers:
	* A simple logging system, to get everything consistent: http://github.com/sqp/godock/libs/log
	* The common applet polling system, helps you handle a regular task: http://github.com/sqp/godock/libs/poller

Cairo-Dock Wiki:
	* DBus methods: http://www.glx-dock.org/ww_page.php?p=Control_your_dock_with_DBus&lang=en
	* DBus for applets: http://www.glx-dock.org/ww_page.php?p=Documentation&lang=en
This last link is mainly the python documentation, but could be considered as
almost valid for Go. It will be imported when the API will get a more stable look.

Some of the actions on the main icon:
	app.SetQuickInfo("OK")
	app.SetLabel("label changed")
	app.SetIcon("/usr/share/icons/gnome/32x32/actions/gtk-media-pause.png")
	app.SetEmblem("/usr/share/icons/gnome/32x32/actions/gtk-go-down.png", cdtype.EmblemTopRight)
	app.Animate("fire", 10)
	app.DemandsAttention(true, "default")
	app.ShowDialog("dialog string\n with time in second", 8)

	app.BindShortkey("<Control><Shift>Y", "<Alt>K")
	app.AddDataRenderer("gauge", 2, "Turbo-night-fuel")
	app.RenderValues(0.2, 0.7})

	app.AskText("Enter your name", "<my name>")
	app.AskValue("How many?", 0, 42)
	app.AskQuestion("Why?")

	app.ControlAppli("devhelp")
	app.ShowAppli(true)

	app.PopulateMenu(items... string) error // only in event BuildMenu
	app.Get(property string) ([]interface{}, error)
	app.GetAll() (*DockProperties, error)

You can add SubIcons:
	app.AddSubIcon(
		"icon 1", "firefox-3.0", "id1",
		"icon 2", "chromium-browser", "id2",
		"icon 3", "geany", "id3",
	)
	app.RemoveSubIcon("id1")

Some of the actions to play with SubIcons:
	app.Icons["id3"].SetQuickInfo("woot")
	app.Icons["id2"].SetLabel("label changed")
	app.Icons["id3"].Animate("fire", 3)

Still to do;
	* DBus Actions missing: PopupDialog, AddMenuItems


Copyright (C) 2012-2014 SQP  <sqp@glx-dock.org>
*/
package godock
