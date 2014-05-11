/*
Package dbus is the godock cairo-dock connector using DBus.

Its goal is to connect the main Cairo-Dock Golang API, godock/libs/dock, to its parent.

Actions on the main icon

Examples:
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

	properties, e := app.GetAll()

	app.PopulateMenu("entry1", "", "after separator") // only in event BuildMenu

Add SubIcons

Some of the actions to play with SubIcons:

	app.AddSubIcon(
		"icon 1", "firefox-3.0",      "id1",
		"text 2", "chromium-browser", "id2",
		"1 more", "geany",            "id3",
	)
	app.RemoveSubIcon("id1")

	app.Icons["id3"].SetQuickInfo("OK")
	app.Icons["id2"].SetLabel("label changed")
	app.Icons["id3"].Animate("fire", 3)

Still to do;
	* Icon Actions missing: PopupDialog, AddMenuItems
*/
package appdbus

import (
	"github.com/guelfey/go.dbus"

	"github.com/sqp/pulseaudio"

	"github.com/sqp/godock/libs/cdtype"

	"errors"
	"reflect"
	"strings"
)

const (
	DbusObject             = "org.cairodock.CairoDock"
	DbusPathDock           = "/org/cairodock/CairoDock"
	DbusInterfaceDock      = "org.cairodock.CairoDock"
	DbusInterfaceApplet    = "org.cairodock.CairoDock.applet"
	DbusInterfaceSubapplet = "org.cairodock.CairoDock.subapplet"
)

// CDDbus is an applet connection to Cairo-Dock using Dbus.
//
type CDDbus struct {
	Icons     map[string]*SubIcon // SubIcons index (by ID).
	Events    cdtype.Events       // Dock events for the icon.
	SubEvents cdtype.SubEvents    // Dock events for subicons.

	Log cdtype.Logger // Applet logger.

	busPath dbus.ObjectPath

	// private data
	dbusIcon *dbus.Object
	dbusSub  *dbus.Object

	hooker *pulseaudio.Hooker
}

// New creates a CDDbus connection.
//
func New(path string) *CDDbus {
	return &CDDbus{
		Icons:   make(map[string]*SubIcon),
		busPath: dbus.ObjectPath(path),
		hooker:  pulseaudio.NewHooker(),
	}
}

//------------------------------------------------------------[ DBUS CONNECT ]--

// SessionBus creates a Dbus session with a listening chan.
//
func SessionBus() (*dbus.Conn, chan *dbus.Signal, error) {
	conn, e := dbus.SessionBus()
	if e != nil {
		return nil, nil, e
	}

	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)
	return conn, c, nil
}

// ConnectToBus connects the applet manager to the dock and register events callbacks.
//
func (cda *CDDbus) ConnectToBus() (<-chan *dbus.Signal, error) {
	conn, c, e := SessionBus()
	if e != nil {
		close(c)
		return nil, e
	}
	return c, cda.ConnectEvents(conn)
}

// ConnectEvents registers to receive Dbus applet events.
//
func (cda *CDDbus) ConnectEvents(conn *dbus.Conn) error {

	cda.hooker.AddCalls(DockCalls)
	cda.hooker.AddTypes(DockTests)

	cda.dbusIcon = conn.Object(DbusObject, cda.busPath)
	cda.dbusSub = conn.Object(DbusObject, cda.busPath+"/sub_icons")
	if cda.dbusIcon == nil || cda.dbusSub == nil {
		return errors.New("no Dbus interface")
	}

	// Listen to all events emitted for the icon.
	matchIcon := "type='signal',path='" + string(cda.busPath) + "',interface='" + DbusInterfaceApplet + "',sender='" + DbusObject + "'"
	matchSubs := "type='signal',path='" + string(cda.busPath) + "/sub_icons',interface='" + DbusInterfaceSubapplet + "',sender='" + DbusObject + "'"

	e := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchIcon).Err
	cda.Log.Err(e, "connect to icon DBus events")
	e = conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchSubs).Err
	cda.Log.Err(e, "connect to subicons DBus events")

	return e
}

// OnSignal forward the received signal to the registered event callback.
// Return true if the signal was quit applet.
//
func (cda *CDDbus) OnSignal(s *dbus.Signal) (exit bool) {
	if s == nil {
		return false
	}

	name := strings.TrimPrefix(string(s.Name), DbusInterfaceApplet+".")
	if name != s.Name { // dbus interface matched.
		if cda.hooker.Call(name, s) { // New method with auto register.
			// return // signal was defined (even if no clients are connected).
		}
		return cda.receivedMainEvent(name, s.Body)
	}

	name = strings.TrimPrefix(string(s.Name), DbusInterfaceSubapplet+".")
	if name != s.Name { // dbus interface matched.
		cda.hooker.Call(name, s)
		cda.receivedSubEvent(name, s.Body)
		return false
	}

	cda.Log.Info("unknown signal", s)
	return false
}

// Call DBus method without returned values.
//
func (cda *CDDbus) launch(iface *dbus.Object, action string, args ...interface{}) error {
	return iface.Call(action, 0, args...).Err
}

func launch(iface *dbus.Object, action string, args ...interface{}) error {
	return iface.Call(action, 0, args...).Err
}

// EavesDrop allow to register to Dbus events for custom parsing.
//
func EavesDrop(match string) (chan *dbus.Message, error) {
	conn, e := dbus.SessionBus()
	if e != nil {
		return nil, e
	}
	e = conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, match).Err
	if e != nil {
		return nil, e
	}
	c := make(chan *dbus.Message, 10)
	conn.Eavesdrop(c)
	return c, nil
}

//
//------------------------------------------------------------[ ICON ACTIONS ]--

// SetQuickInfo change the quickinfo text displayed on the icon.
//
func (cda *CDDbus) SetQuickInfo(info string) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".SetQuickInfo", info)
}

// SetLabel change the text label next to the icon.
//
func (cda *CDDbus) SetLabel(label string) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".SetLabel", label)
}

// SetIcon set the image of the icon, overwriting the previous one.
// A lot of image formats are supported, including SVG.
// You can refer to the image by either its name if it's an image from a icon theme, or by a path.
//   app.SetIcon("gimp")
//   app.SetIcon("gtk-go-up")
//   app.SetIcon("/path/to/image")
//
func (cda *CDDbus) SetIcon(icon string) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".SetIcon", icon)
}

// SetEmblem set an emblem image on the icon. To remove it, you have to use
// SetEmblem again with an empty string.
//
//   app.SetEmblem(app.FileLocation("img", "emblem-work.png"), cdtype.EmblemBottomLeft)
//
func (cda *CDDbus) SetEmblem(icon string, position cdtype.EmblemPosition) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".SetEmblem", icon, int32(position))
}

// Animate animates the icon, with a given animation and for a given number of
// rounds.
//
func (cda *CDDbus) Animate(animation string, rounds int32) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".Animate", animation, rounds)
}

// DemandsAttention is like the Animate method, but will animate the icon
// endlessly, and the icon will be visible even if the dock is hidden. If the
// animation is an empty string, or "default", the animation used when an
// application demands the attention will be used.
// The first argument is true to start animation, or false to stop it.
//
func (cda *CDDbus) DemandsAttention(start bool, animation string) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".DemandsAttention", start, animation)
}

// ShowDialog pops up a simple dialog bubble on the icon.
// The dialog can be closed by clicking on it.
//
func (cda *CDDbus) ShowDialog(message string, duration int32) error {
	return cda.dbusIcon.Go(DbusInterfaceApplet+".ShowDialog", 0, nil, message, duration).Err
}

// PopupDialog open a dialog box . The dialog can contain a message, an icon,
// some buttons, and a widget the user can act on.
//
// Adding buttons will trigger an on_answer_dialog signal when the user press
// one of them. "ok" and "cancel" are used as keywords defined by the dock.
//
// Dialog attributes:
//   message        string    dialog box text (default=empty).
//   icon           string    icon displayed next to the message (default=applet icon).
//   time-length    bool      duration of the dialog, in second (default=unlimited).
//   force-above    bool      true to force the dialog above. Use it with parcimony (default=false)
//   use-markup     bool      true to use Pango markup to add text decorations (default=false).
//   buttons        string    images of the buttons, separated by comma ";" (default=none).
//
// Widget attributes:
//   type          string    type of the widget: "text-entry" or "scale" or "list".
//
// Widget text-entry attributes:
//   multi-lines    bool      true to have a multi-lines text-entry, ie a text-view (default=false).
//   editable       bool      whether the user can modify the text or not (default=true).
//   visible        bool      whether the text will be visible or not (useful to type passwords) (default=true).
//   nb-chars       int32     maximum number of chars (the current number of chars will be displayed next to the entry) (default=infinite).
//   initial-value  string    text initially contained in the entry (default=empty).
//
// Widget scale attributes:
//   min-value      double    lower value (default=0).
//   max-value      double    upper value (default=100).
//   nb-digit       int32     number of digits after the dot (default=2).
//   initial-value  double    value initially set to the scale (default=0).
//   min-label      string    label displayed on the left of the scale (default=empty).
//   max-label      string    label displayed on the right of the scale (default=empty).
//
// Widget list attributes:
//   editable       bool      true if a non-existing choice can be entered by the user (in this case, the content of the widget will be the selected text, and not the number of the selected line) (false by default)
//   values         string    a list of values, separated by comma ";", used to fill the combo list.
//   initial-value  string or int32 depending on the "editable" attribute :
//        case editable=true:   string with the default text for the user entry of the widget (default=empty).
//        case editable=false:  int with the selected line number (default=0).
//
func (cda *CDDbus) PopupDialog(dialog map[string]interface{}, widget map[string]interface{}) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".PopupDialog", toMapVariant(dialog), toMapVariant(widget))
}

// AddDataRenderer add a graphic data renderer to the icon.
//
//  Renderer types: gauge, graph, progressbar.
//  Themes for renderer Graph: "Line", "Plain", "Bar", "Circle", "Plain Circle"
//
func (cda *CDDbus) AddDataRenderer(typ string, nbval int32, theme string) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".AddDataRenderer", typ, nbval, theme)
}

// RenderValues render new values on the icon.
//   * You must have added a data renderer before with AddDataRenderer.
//   * The number of values sent must match the number declared before.
//   * Values are given between 0 and 1.
//
func (cda *CDDbus) RenderValues(values ...float64) error {
	// return cda.dbusIcon.Call("RenderValues", dbus.FlagNoAutoStart, values).Err
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".RenderValues", values)
}

// ActOnAppli send an action on the application controlled by the icon (see ControlAppli).
//
//   "minimize"            to hide the window
//   "show"                to show the window and give it focus
//   "toggle-visibility"   to show or hide
//   "maximize"            to maximize the window
//   "restore"             to restore the window
//   "toggle-size"         to maximize or restore
//   "close"               to close the window (Note: some programs will just hide the window and stay in the systray)
//   "kill"                to kill the X window
//
func (cda *CDDbus) ActOnAppli(action string) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".ActOnAppli", action)
}

// ControlAppli allow your applet to control the window of an external
// application and can steal its icon from the Taskbar.
//  *Use the xprop command find the class of the window you want to control.
//  *Use "none" if you want to reset application control.
//  *Controling an application enables the OnFocusChange callback.
//
func (cda *CDDbus) ControlAppli(applicationClass string) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".ControlAppli", applicationClass)
}

// ShowAppli set the visible state of the application controlled by the icon.
//
func (cda *CDDbus) ShowAppli(show bool) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".ShowAppli", interface{}(show))
}

// aa{sv}

//~ func (cda *CDDbus) AddMenuItems(items... []map[string]interface{}) error {

// AddMenuItems is broken, sorry TODO.
//
func (cda *CDDbus) AddMenuItems() error {
	menuitem := []map[string]interface{}{
		{"widget-type": cdtype.MenuEntry, //int32(0),
			"label": "entry",
			// "icon":  "gtk-add",
			"menu": int32(0),
			"id":   int32(1),
			// "tooltip": "this is the tooltip that will appear when you hover this entry",
		},
		// {},
	}

	var data []map[string]dbus.Variant
	for _, interf := range menuitem {
		data = append(data, toMapVariant(interf))
	}

	// icon := map[string]dbus.Variant{

	// "widget-type": dbus.MakeVariant(int32(cdtype.MenuEntry)),
	// "label":       dbus.MakeVariant("this is an entry of the main menu"),
	// "icon":  dbus.MakeVariant("gtk-add"),
	// "menu":    int32(0),
	// "id":      int32(1),
	// "tooltip": "this is the tooltip that will appear when you hover this entry",
	// }

	// cda.Log.Info("struct", data)
	// cda.Log.Err(cda.launch(cda.dbusIcon, DbusInterfaceApplet+".AddMenuItems", data), "AddMenuItems")
	// cda.Log.Err(cda.dbusIcon.Call(DbusInterfaceApplet+".AddMenuItems", 0, data).Err, "additems")
	cda.Log.NewErr("Disabled, prevent a crash")
	return nil
}

// PopulateMenu adds a list of entry to the default menu. An empty string will
// add a separator. Can only be used in the OnBuildMenu callback.
//
func (cda *CDDbus) PopulateMenu(items ...string) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".PopulateMenu", items)
}

// BindShortkey binds one or more keyboard shortcuts to your applet. Only non
// empty shortkeys will be sent to the dock so you can use this method to
// directly add them from config.
//
func (cda *CDDbus) BindShortkey(shortkeys ...string) error {
	return cda.launch(cda.dbusIcon, DbusInterfaceApplet+".BindShortkey", shortkeys)
}

// AskText pops up a dialog with a text entry to ask user feedback.
// The answer will be forwarded with the OnAnswerDialog callback.
//
func (cda *CDDbus) AskText(message, initialText string) error {
	return cda.dbusIcon.Call("AskText", 0, message, initialText).Err
}

// AskValue pops up a dialog with a slider to get user feedback.
// The answer will be forwarded with the OnAnswerDialog callback.
//
func (cda *CDDbus) AskValue(message string, initialValue, maxValue float64) error {
	return cda.dbusIcon.Call("AskValue", 0, message, initialValue, maxValue).Err
}

// AskQuestion need more documentation TODO.
//
func (cda *CDDbus) AskQuestion(message string) error {
	return cda.dbusIcon.Call("AskQuestion", 0, message).Err
}

// Get a property of the icon of your applet. Current available properties are :
//   x            int32     x position of the icon's center on the screen (starting from 0 on the left)
//   y            int32     y position of the icon's center on the screen (starting from 0 at the top of the screen)
//   width        int32     width of the icon, in pixels (this is the maximum width, when the icon is zoomed)
//   height       int32     height of the icon, in pixels (this is the maximum height, when the icon is zoomed)
//   container    uint32   type of container of the applet (DOCK, DESKLET)
//   orientation  uint32   position of the container on the screen (BOTTOM, TOP, RIGHT, LEFT). A desklet has always an orientation of BOTTOM.
//   Xid          uint64   ID of the application's window which is controlled by the applet, or 0 if none (this parameter can only be non nul if you used the method ControlAppli beforehand).
//   has_focus    bool     Whether the application's window which is controlled by the applet is the current active window (it has the focus) or not. E.g.:
//
func (cda *CDDbus) Get(property string) (interface{}, error) {
	var v dbus.Variant
	e := cda.dbusIcon.Call("Get", 0, property).Store(&v)
	return v.Value(), e
}

// GetAll returns applet icon properties.
//
func (cda *CDDbus) GetAll() *cdtype.DockProperties {
	vars := make(map[string]dbus.Variant)
	if cda.Log.Err(cda.dbusIcon.Call("GetAll", 0).Store(&vars), "dbus GetAll") {
		return nil
	}

	props := &cdtype.DockProperties{}
	for k, v := range vars {
		switch k {
		case "Xid":
			props.Xid = v.Value().(uint64)
		case "x":
			props.X = v.Value().(int32)
		case "y":
			props.Y = v.Value().(int32)
		case "orientation":
			props.Orientation = v.Value().(uint32)
		case "container":
			props.Container = v.Value().(uint32)
		case "width":
			props.Width = v.Value().(int32)
		case "height":
			props.Height = v.Value().(int32)
		case "has_focus":
			props.HasFocus = v.Value().(bool)
		}
	}
	return props
}

//
//--------------------------------------------------------[ SUBICONS ACTIONS ]--

// AddSubIcon adds subicons by pack of 3 strings : label, icon, id.
//
func (cda *CDDbus) AddSubIcon(fields ...string) error {
	for i := 0; i < len(fields)/3; i++ {
		id := fields[3*i+2]
		cda.Icons[id] = &SubIcon{cda.dbusSub, id}
	}
	return cda.launch(cda.dbusSub, DbusInterfaceSubapplet+".AddSubIcons", fields)
}

// RemoveSubIcon only need the ID to remove the SubIcon.
//
func (cda *CDDbus) RemoveSubIcon(id string) error {
	if _, ok := cda.Icons[id]; !ok {
		return errors.New("RemoveSubIcon Icon missing: " + id)
	}

	e := cda.launch(cda.dbusSub, DbusInterfaceSubapplet+".RemoveSubIcon", id)
	if e == nil {
		delete(cda.Icons, id)
	}
	return e
}

// SubIcon defines a connection to the subdock icon.
//
type SubIcon struct {
	dbusSub *dbus.Object
	id      string
}

// SetQuickInfo change the quickinfo text displayed on the subicon.
//
func (cdi *SubIcon) SetQuickInfo(info string) error {
	return launch(cdi.dbusSub, DbusInterfaceSubapplet+".SetQuickInfo", info, cdi.id)
}

// SetLabel change the text label next to the subicon.
//
func (cdi *SubIcon) SetLabel(label string) error {
	return launch(cdi.dbusSub, DbusInterfaceSubapplet+".SetLabel", label, cdi.id)
}

// SetIcon set the image of the subicon, overwriting the previous one. See Icon.
//
func (cdi *SubIcon) SetIcon(icon string) error {
	return launch(cdi.dbusSub, DbusInterfaceSubapplet+".SetIcon", icon, cdi.id)
}

// SetEmblem set an emblem image on the subicon. See Icon.
//
func (cdi *SubIcon) SetEmblem(icon string, position cdtype.EmblemPosition) error {
	return launch(cdi.dbusSub, DbusInterfaceSubapplet+".SetEmblem", icon, int32(position), cdi.id)
}

// Animate animates the subicon, with a given animation and for a given number of
// rounds. See Icon.
//
func (cdi *SubIcon) Animate(animation string, rounds int32) error {
	return launch(cdi.dbusSub, DbusInterfaceSubapplet+".Animate", animation, rounds, cdi.id)
}

// ShowDialog pops up a simple dialog bubble on the subicon. See Icon.
//
func (cdi *SubIcon) ShowDialog(message string, duration int32) error {
	return launch(cdi.dbusSub, DbusInterfaceSubapplet+".ShowDialog", message, duration, cdi.id)
}

//
//----------------------------------------------------------[ EVENT CALLBACK ]--

// Event receiver, dispatch it to the configured callback.
//
func (cda *CDDbus) receivedMainEvent(event string, data []interface{}) (exit bool) {
	switch event {
	case "on_stop_module":
		cda.Log.Debug("Received from dock", event)
		if cda.Events.End != nil {
			cda.Events.End()
		}
		return true

	case "on_reload_module":
		if cda.Events.Reload != nil {
			go cda.Events.Reload(data[0].(bool))
		}
	case "on_click":
		if cda.Events.OnClick != nil {
			go cda.Events.OnClick()
		}
	case "on_middle_click":
		if cda.Events.OnMiddleClick != nil {
			go cda.Events.OnMiddleClick()
		}
	case "on_build_menu":
		if cda.Events.OnBuildMenu != nil {
			go cda.Events.OnBuildMenu()
		}
	case "on_menu_select":
		if cda.Events.OnMenuSelect != nil {
			go cda.Events.OnMenuSelect(data[0].(int32))
		}
	case "on_scroll":
		if cda.Events.OnScroll != nil {
			go cda.Events.OnScroll(data[0].(bool))
		}
	case "on_drop_data":
		if cda.Events.OnDropData != nil {
			go cda.Events.OnDropData(data[0].(string))
		}
	case "on_answer":
		if cda.Events.OnAnswer != nil {
			go cda.Events.OnAnswer(data[0])
		}
	case "on_answer_dialog":
		if cda.Events.OnAnswerDialog != nil {
			go cda.Events.OnAnswerDialog(data[0].(int32), data[1])
		}
	case "on_shortkey":
		if cda.Events.OnShortkey != nil {
			go cda.Events.OnShortkey(data[0].(string))
		}
	case "on_change_focus":
		if cda.Events.OnChangeFocus != nil {
			go cda.Events.OnChangeFocus(data[0].(bool))
		}
	default:
		cda.Log.Info("unknown icon event", event, data)
	}
	return false
}

func (cda *CDDbus) receivedSubEvent(event string, data []interface{}) {
	switch event {
	case "on_click_sub_icon":
		if cda.SubEvents.OnSubClick != nil {
			go cda.SubEvents.OnSubClick(data[0].(int32), data[1].(string))
		}
	case "on_middle_click_sub_icon":
		if cda.SubEvents.OnSubMiddleClick != nil {
			go cda.SubEvents.OnSubMiddleClick(data[0].(string))
		}
	case "on_scroll_sub_icon":
		if cda.SubEvents.OnSubScroll != nil {
			go cda.SubEvents.OnSubScroll(data[0].(bool), data[1].(string))
		}
	case "on_drop_data_sub_icon":
		if cda.SubEvents.OnSubDropData != nil {
			go cda.SubEvents.OnSubDropData(data[0].(string), data[1].(string))
		}
	case "on_build_menu_sub_icon":
		if cda.SubEvents.OnSubBuildMenu != nil {
			go cda.SubEvents.OnSubBuildMenu(data[0].(string))
		}
	default:
		cda.Log.Info("unknown subicon event", event, data)
	}
}

//
//------------------------------------------------------------------[ COMMON ]--

// Recast list of args to map[string]dbus.Variant as requested by the DBus API.
//
func toMapVariant(input map[string]interface{}) map[string]dbus.Variant {
	vars := make(map[string]dbus.Variant)
	for k, v := range input {
		vars[k] = dbus.MakeVariant(v)
	}
	return vars
}

//
//-----------------------------------------------------------[ NEW CALLBACKS ]--

// Register connects an object to the dock events hooks it implements.
// If the object declares any of the method in the Define... interfaces list, it
// will be registered to receive those events.
//
func (cda *CDDbus) RegisterEvents(obj interface{}) (errs []error) {
	tolisten := cda.hooker.Register(obj)
	cda.Log.Debug("listened events", tolisten)
	return errs
}

func (cda *CDDbus) UnregisterEvents(obj interface{}) (errs []error) {
	// tolisten := cda.hooker.Unregister(obj)
	// cda.Log.Info("toliste", tolisten)
	// _ = tolisten
	return errs
}

type DefineOnClick interface {
	OnClick()
}

type DefineOnMiddleClick interface {
	OnMiddleClick()
}

type DefineOnBuildMenu interface {
	OnBuildMenu()
}

type DefineOnMenuSelect interface {
	OnMenuSelect(int32)
}

type DefineOnScroll interface {
	OnScroll(up bool)
}

type DefineOnDropData interface {
	OnDropData(string)
}

type DefineOnAnswer interface {
	OnAnswer(interface{})
}

type DefineOnAnswerDialog interface {
	OnAnswerDialog(int32, interface{})
}

type DefineOnShortkey interface {
	OnShortkey(string)
}

type DefineOnChangeFocus interface {
	OnChangeFocus(bool)
}

type DefineOnReload interface {
	OnReload(bool)
}

type DefineOnStopModule interface {
	OnStopModule()
}

type DefineOnSubClick interface {
	OnSubClick(int32, string)
}

type DefineOnSubMiddleClick interface {
	OnSubMiddleClick(string)
}

type DefineOnSubScroll interface {
	OnSubScroll(bool, string)
}

type DefineOnSubDropData interface {
	OnSubDropData(string, string)
}

type DefineOnSubBuildMenu interface {
	OnSubBuildMenu(string)
}

//
//--------------------------------------------------------[ CALLBACK METHODS ]--

// DockCalls defines callbacks methods for matching objects with type-asserted arguments.
// Public so it can be hacked before the first Register.
//
var DockCalls = pulseaudio.Calls{
	"on_click":         func(m pulseaudio.Msg) { m.O.(DefineOnClick).OnClick() },
	"on_middle_click":  func(m pulseaudio.Msg) { m.O.(DefineOnMiddleClick).OnMiddleClick() },
	"on_build_menu":    func(m pulseaudio.Msg) { m.O.(DefineOnBuildMenu).OnBuildMenu() },
	"on_menu_select":   func(m pulseaudio.Msg) { m.O.(DefineOnMenuSelect).OnMenuSelect(m.D[0].(int32)) },
	"on_scroll":        func(m pulseaudio.Msg) { m.O.(DefineOnScroll).OnScroll(m.D[0].(bool)) },
	"on_drop_data":     func(m pulseaudio.Msg) { m.O.(DefineOnDropData).OnDropData(m.D[0].(string)) },
	"on_answer":        func(m pulseaudio.Msg) { m.O.(DefineOnAnswer).OnAnswer(m.D[0]) },                             // type 0 unknown, to improve
	"on_answer_dialog": func(m pulseaudio.Msg) { m.O.(DefineOnAnswerDialog).OnAnswerDialog(m.D[0].(int32), m.D[1]) }, // type 1 unknown, to improve
	"on_shortkey":      func(m pulseaudio.Msg) { m.O.(DefineOnShortkey).OnShortkey(m.D[0].(string)) },
	"on_change_focus":  func(m pulseaudio.Msg) { m.O.(DefineOnChangeFocus).OnChangeFocus(m.D[0].(bool)) },
	"on_reload_module": func(m pulseaudio.Msg) { m.O.(DefineOnReload).OnReload(m.D[0].(bool)) },
	"on_stop_module":   func(m pulseaudio.Msg) { m.O.(DefineOnStopModule).OnStopModule() },

	"on_click_sub_icon":        func(m pulseaudio.Msg) { m.O.(DefineOnSubClick).OnSubClick(m.D[0].(int32), m.D[1].(string)) },
	"on_middle_click_sub_icon": func(m pulseaudio.Msg) { m.O.(DefineOnSubMiddleClick).OnSubMiddleClick(m.D[0].(string)) },
	"on_scroll_sub_icon":       func(m pulseaudio.Msg) { m.O.(DefineOnSubScroll).OnSubScroll(m.D[0].(bool), m.D[1].(string)) },
	"on_drop_data_sub_icon":    func(m pulseaudio.Msg) { m.O.(DefineOnSubDropData).OnSubDropData(m.D[0].(string), m.D[1].(string)) },
	"on_build_menu_sub_icon":   func(m pulseaudio.Msg) { m.O.(DefineOnSubBuildMenu).OnSubBuildMenu(m.D[0].(string)) },
}

// DockTests defines callbacks to test if objects are implementing the callback interface.
// Public so it can be hacked before the first Register.
//
var DockTests = pulseaudio.Types{
	"on_click":         reflect.TypeOf((*DefineOnClick)(nil)).Elem(),
	"on_middle_click":  reflect.TypeOf((*DefineOnMiddleClick)(nil)).Elem(),
	"on_build_menu":    reflect.TypeOf((*DefineOnBuildMenu)(nil)).Elem(),
	"on_menu_select":   reflect.TypeOf((*DefineOnMenuSelect)(nil)).Elem(),
	"on_scroll":        reflect.TypeOf((*DefineOnScroll)(nil)).Elem(),
	"on_drop_data":     reflect.TypeOf((*DefineOnDropData)(nil)).Elem(),
	"on_answer":        reflect.TypeOf((*DefineOnAnswer)(nil)).Elem(),
	"on_answer_dialog": reflect.TypeOf((*DefineOnAnswerDialog)(nil)).Elem(),
	"on_shortkey":      reflect.TypeOf((*DefineOnShortkey)(nil)).Elem(),
	"on_change_focus":  reflect.TypeOf((*DefineOnChangeFocus)(nil)).Elem(),
	"on_reload_module": reflect.TypeOf((*DefineOnReload)(nil)).Elem(),
	"on_stop_module":   reflect.TypeOf((*DefineOnStopModule)(nil)).Elem(),

	"on_click_sub_icon":        reflect.TypeOf((*DefineOnSubClick)(nil)).Elem(),
	"on_middle_click_sub_icon": reflect.TypeOf((*DefineOnSubMiddleClick)(nil)).Elem(),
	"on_scroll_sub_icon":       reflect.TypeOf((*DefineOnSubScroll)(nil)).Elem(),
	"on_drop_data_sub_icon":    reflect.TypeOf((*DefineOnSubDropData)(nil)).Elem(),
	"on_build_menu_sub_icon":   reflect.TypeOf((*DefineOnSubBuildMenu)(nil)).Elem(),
}

//
//---------------------------------------------------------[ UNUSED / BUGGED ]--

/*


	// Connect defined events callbacks.
	// typ := reflect.TypeOf(cda.Events)
	// elem := reflect.ValueOf(&cda.Events).Elem()
	// for i := 0; i < typ.NumField(); i++ { // Parsing all fields in type.
	// 	cda.connectEvent(elem.Field(i), typ.Field(i))
	// }


// Connect an event to the dock if a callback is defined.
//
func (cda *CDDbus) connectEvent(elem reflect.Value, structField reflect.StructField) {
	conn, _ := dbus.SessionBus()

	tag := structField.Tag.Get("event")                          // Field must have the event tag.
	if tag != "" && (!elem.IsNil() || tag == "on_stop_module") { // And a valid callback. stop module is mandatory for the close signal.
		log.Info("Binded event", tag)
		// 	rule := &dbus.MatchRule{
		// 		Type:      dbus.TypeSignal,
		// 		Interface: DbusInterfaceApplet,
		// 		Member:    tag,
		// 		Path:      cda.busPath,

		var ret interface{}
		e := conn.BusObject().Call(
			"org.freedesktop.DBus.AddMatch",
			0,
			// "type='signal',sender='org.freedesktop.DBus'").Store()
			"type='signal',path='"+string(cda.busPath)+"',interface='"+DbusInterfaceApplet+"',sender='"+DbusObject+"'").Store()
		log.DEV("omar", ret, e)
	}

	// 	cda.dbus.Handle(rule, func(msg *dbus.Message) { cda.receivedMainEvent(msg) })
	// }
}
*/

/*
func (cda *CDDbus) GetIconProperties() interface{} {
	base := cda.dbus.Object("org.cairodock.CairoDock", "/org/cairodock/CairoDock").Interface("org.cairodock.CairoDock")
	//~ return cda.call(base, "GetIconProperties", "container=_MainDock_")
	return cda.call(base, "GetIconProperties", interface{}("class=chromium-browser"))
	//~ return cda.call(base, "GetIconProperties")
}

func (cda *CDDbus) GetContainerProperties() []interface{} {
	//~ props := &DockProperties{}

	base := cda.dbus.Object("org.cairodock.CairoDock", "/org/cairodock/CairoDock").Interface("org.cairodock.CairoDock")
	data, _ := cda.call(base, "GetContainerProperties", "_MainDock_")
return data
	//~ var args []interface{}{}:= interface{}("_MainDock_")
	//~ args := []string{"_MainDock_"}
	//~ args := "_MainDock_"
	//~ return cda.call(base, "GetIconProperties", "container=_MainDock_")
	//~ return cda.call(base, "GetContainerProperties", "_MainDock_", "")
	//~ return cda.call(base, "GetIconProperties")
}


*/

// func call(connect *dbus.Connection, iface *dbus.Interface, action string, args ...interface{}) error {
// 	if iface == nil {
// 		return errors.New("no subicon interface")
// 	}
// 	method, e := iface.Method(action)
// 	if e != nil {
// 		return e
// 	}
// 	_, err := connect.Call(method, args...)
// 	//~ fmt.Println("ret", ret)
// 	return err
// }

// func dbusAsync(connect *dbus.Connection, iface *dbus.Interface, action string, args ...interface{}) error {
// 	if iface == nil {
// 		return errors.New("no subicon interface")
// 	}
// 	method, e := iface.Method(action)
// 	if e != nil {
// 		return e
// 	}
// 	return connect.CallAsync(method, args...)
// }
