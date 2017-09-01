// Package Notifications is a desktop notifications history applet for Cairo-Dock.
//
// requires a hacked version of the dbus api (that wont stop after eavesdropping a message).
//
package Notifications

// https://developer.gnome.org/notification-spec/

import (
	"github.com/godbus/dbus"

	"github.com/sqp/godock/libs/cdtype"             // Applet types.
	"github.com/sqp/godock/libs/srvdbus/dbuscommon" // EavesDrop

	"strconv"
	"strings"
)

//
//------------------------------------------------------------------[ APPLET ]--

func init() { cdtype.Applets.Register("Notifications", NewApplet) }

// Applet defines a dock applet.
//
type Applet struct {
	cdtype.AppBase // Applet base and dock connection.

	conf   *appletConf
	notifs *Notifs
}

// NewApplet creates a new applet instance.
//
func NewApplet(base cdtype.AppBase, events *cdtype.Events) cdtype.AppInstance {
	app := &Applet{AppBase: base, notifs: &Notifs{log: base.Log()}}
	app.SetConfig(&app.conf, app.actions()...)

	// Events.
	events.OnClick = app.Action().Callback(ActionShowAll)
	events.OnMiddleClick = app.Action().Callback(ActionClear)
	events.OnBuildMenu = app.Action().CallbackMenu(menuUser...)

	// Notifs.
	app.notifs.SetOnCount(app.UpdateCount)
	e := app.notifs.Start()
	app.Log().Err(e, "notifications listener")

	return app
}

// Init loads user configuration if needed and initialise applet.
//
func (app *Applet) Init(def *cdtype.Defaults, confLoaded bool) {
	// Set notification service config.
	app.notifs.NotifConfig = app.conf.NotifConfig

	// New message icon.
	if app.conf.NotifAltIcon == "" {
		app.conf.NotifAltIcon = app.FileLocation(defaultNotifAltIcon)
	}
}

//
//-----------------------------------------------------------------[ ACTIONS ]--

// Define applet actions. Order must match actions const declaration order.
//
func (app *Applet) actions() []*cdtype.Action {
	return []*cdtype.Action{
		{
			ID:   ActionNone,
			Menu: cdtype.MenuSeparator,
		}, {
			ID:       ActionShowAll,
			Name:     "Show messages",
			Icon:     "media-seek-forward",
			Call:     app.displayAll,
			Threaded: true,
		}, {
			ID:       ActionClear,
			Name:     "Clear all",
			Icon:     "edit-clear",
			Call:     app.notifs.Clear,
			Threaded: true,
		},
	}
}

//
//-----------------------------------------------------------------[ DISPLAY ]--

// UpdateCount shows the number of messages on the icon, and displays the
// alternate icon if count > 0.
//
func (app *Applet) UpdateCount(count int) {
	text := ""
	icon := ""
	switch {
	case count > 0:
		icon = app.conf.NotifAltIcon
		text = strconv.Itoa(count)

	case app.conf.Icon != "":
		icon = app.conf.Icon
	}
	app.SetQuickInfo(text)
	app.SetIcon(icon)
}

func (app *Applet) displayAll() {
	var msg string
	messages := app.notifs.List()
	if len(messages) == 0 {
		msg = "No recent notifications"

	} else {
		text, e := app.conf.DialogTemplate.ToString("ListNotif", messages)
		app.Log().Err(e, "template")
		msg = strings.TrimRight(text, "\n")
	}

	app.PopupDialog(cdtype.DialogData{
		Message:    msg,
		UseMarkup:  true,
		Buttons:    "edit-clear;cancel",
		TimeLength: app.conf.DialogDuration,
		Callback:   cdtype.DialogCallbackValidNoArg(app.Action().Callback(ActionClear)), // Clear notifs if the user press the 1st button.
	})

	// if self.config['clear'] else 4 + len(msg)/40 }  // if we're going to clear the history, show the dialog until the user closes it
}

//
//-----------------------------------------------------------[ NOTIFICATIONS ]--

// Notif defines a single Dbus notification.
//
type Notif struct {
	Sender, Icon, Title, Content string
	duration, ID                 uint32
}

// Notifs handles Dbus notifications management.
//
type Notifs struct {
	NotifConfig

	C chan *dbus.Message

	messages  []*Notif
	callCount func(int)
	log       cdtype.Logger
}

// NotifConfig defines the notification service configuration.
//
type NotifConfig struct {
	MaxSize   int
	Blacklist []string
}

const match = "type='method_call',path='/org/freedesktop/Notifications',member='Notify',eavesdrop='true'"

// List returns the list of notifications.
//
func (notifs *Notifs) List() []*Notif {
	return notifs.messages
}

// Clear resets the list of notifications.
//
func (notifs *Notifs) Clear() {
	notifs.messages = nil
	if notifs.callCount != nil {
		notifs.callCount(len(notifs.messages))
	}
}

// Add a new notifications to the list.
//
func (notifs *Notifs) Add(newtif *Notif) {
	if newtif == nil {
		return
	}

	for _, ignore := range notifs.Blacklist {
		if newtif.Sender == ignore {
			return
		}
	}

	if !notifs.replace(newtif) {
		notifs.messages = append(notifs.messages, newtif)
		if len(notifs.messages) > notifs.MaxSize {
			notifs.messages = notifs.messages[len(notifs.messages)-notifs.MaxSize:]
		}
	}

	if notifs.callCount != nil {
		notifs.callCount(len(notifs.messages))
	}
}

// try to replace an old notification (same id). Return true if replaced.
//
func (notifs *Notifs) replace(newtif *Notif) bool {
	// removed for now, ID was always 0.
	// for i, oldtif := range notifs.messages {
	// if oldtif.ID == newtif.ID {

	// 	// TODO:REMOVE !!!
	// 	println("replaced", oldtif.ID, newtif.ID)

	// 	notifs.messages[i] = newtif
	// 	return true
	// }
	// }
	return false
}

// SetOnCount sets the callback for notifications count change.
//
func (notifs *Notifs) SetOnCount(call func(int)) {
	notifs.callCount = call
}

// Start the message eavesdropping loop and forward notifs changes to the callback.
//
func (notifs *Notifs) Start() error {
	var e error
	notifs.C, e = dbuscommon.EavesDrop(match)
	if e != nil {
		return e
	}
	notifs.log.GoTry(notifs.Listen)
	return nil
}

// Listen to eavesdropped messages to find notifications..
//
func (notifs *Notifs) Listen() {
	for msg := range notifs.C {
		switch msg.Type {
		// case dbus.TypeSignal:

		case dbus.TypeMethodCall:
			// ensure we got a valid message
			if msg.Headers[dbus.FieldMember].Value().(string) == "Notify" && len(msg.Body) >= 8 {
				notifs.Add(messageToNotif(msg))
			}
		}
	}
}

// messageToNotif converts the dbus message to a notification.
//
func messageToNotif(message *dbus.Message) *Notif {
	newtif := &Notif{
		Sender:  message.Body[0].(string),
		ID:      message.Body[1].(uint32), // always 0 ??
		Icon:    message.Body[2].(string),
		Title:   message.Body[3].(string),
		Content: message.Body[4].(string),
		// duration: message.Body[7],
	}

	// Title too short (it's probably something we don't mind, like a notification that the volume has changed)
	if len(newtif.Title) < 2 {
		return nil
	}

	return newtif
}
