package appdbus

import (
	"github.com/godbus/dbus"

	"github.com/sqp/godock/libs/log"
	"github.com/sqp/godock/libs/packages"

	"strings"
)

var busD *dbus.Object

func busDock() *dbus.Object {
	if busD == nil {
		conn, err := dbus.SessionBus()
		if err != nil {
			log.Info("DBus Connect", err)
			return nil
		}
		busD = conn.Object(DbusObject, DbusPathDock)
	}
	return busD
}

func dockCall(method string, args ...interface{}) error {
	return busDock().Call(DbusInterfaceDock+"."+method, 0, args...).Err
}

//------------------------------------------------------------[ DOCK ACTIONS ]--

// DockAdd adds an item to the Dock.
//
//   Launcher from desktop file:      "type":"Launcher", "config-file":"application://vlc.desktop"
//   Launcher custom:                 "type":"Launcher", "name":"Top 10", "command":"xterm -e top", "icon":"gtk-home.png"
//   Stack icon (SubDock container)   "type":"Stack-Icon", "name":"my sub-dock", "icon":"folder.png"
//   Separator                        "type":"Separator"
//   Module                           "type":"Module", "module":"clock"
//   MainDock                         "type":"Dock"
//
// Optional arguments:
//   Icon relative position                   "order":5
//   Icon location (main or subdock name)     "container":"_MainDock_"
//   Launcher application class               "class":"gjiten"
//
func DockAdd(args map[string]interface{}) error {
	return dockCall("Add", toMapVariant(args))
}

// DockRemove removes an item from the Dock.
//
//   Launcher                                  "type=Launcher & class=vlc"
//   Second main dock (and all its content)    "type=Dock & name=_MainDock_-2"
//   Module                                    "type=Module & name=clock"
//   Instance of a module                      "type=Module-Instance & config-file=clock.conf"
//
func DockRemove(arg string) error {
	return dockCall("Remove", arg)
}

// func (cda *CdDbus) ActivateModule(module string, state bool) {
// 	busDock().Call(DbusInterfaceDock+".ActivateModule", 0, module, state)
// 	// return cda.launch(base, "ActivateModule", interface{}(module), interface{}(state))
// }

// DockReboot reload the current theme of the Dock, as if you had quitted and restarted the dock.
//
func DockReboot() error {
	return dockCall("Reboot")
}

// DockQuit closes the Dock program.
//
func DockQuit() error {
	return dockCall("Quit")
}

// DockShow sets the dock visibility: 0 = HIDE, 1 = SHOW, 2 = TOGGLE.
// If you have several docks, it will show/hide all of them.
//
func DockShow(mode int32) error {
	return dockCall("ShowDock", mode)
}

// ShowDesklet TODO: need to complete this part.
//
func ShowDesklet(mode int32) error {
	return dockCall("ShowDeslet", mode)
}

// IconReload reloads an icon settings from disk.
//
//   "type=Module & name=weather"
//   "config-file=full_path_to_config_or_desktop_file"
//
func IconReload(arg string) error {
	return dockCall("Reload", arg)
}

// DockProperties gets properties of different parts of the dock.
// API may change for this function. Need to figure out the best way to return the data.
//
//   "type=Launcher & class=firefox"
//   "type=Module"
//   "type=Desklet"
//
//   var name, icon string
//   for _, t := range vars {
//   	for k, v := range t {
//   		if k == "icon" {
//   			log.Info(mod, v)
//   		}
//   	}
//   }
func DockProperties(arg string) (vars []map[string]dbus.Variant) {
	log.Err(busDock().Call("GetProperties", 0, arg).Store(&vars), "dbus GetProperties")
	return
}

//--------------------------------------------------[ GET SPECIAL PROPERTIES ]--

// AppletAdd adds an applet instance referenced by its name.
//
func AppletAdd(name string) error {
	return DockAdd(map[string]interface{}{"type": "Module", "module": name})
}

// AppletRemove removes an applet instance referenced by its config file.
//
func AppletRemove(configFile string) error {
	return DockRemove("type=Module-Instance & config-file=" + configFile)
}

// AppletInstances asks the dock details about an applet.
//
func AppletInstances(name string) []string {
	query := "type=Module & name=" + strings.Replace(name, "-", " ", -1)
	if vars := DockProperties(query); len(vars) > 0 {
		if val, ok := vars[0]["instances"]; ok {
			if instances, ok := val.Value().([]string); ok {
				return instances
			}
		}
	}

	// for _, props := range vars {
	// 	for k, v := range props {
	// 		// if k == "name" {
	// 		// 	log.Info(v.String())
	// 		// }
	// 		if k == "instances" {
	// 			if instances, ok := v.Value().([]string); ok {
	// 				return instances
	// 				log.Info("", instances)
	// 			}
	// 		}
	// 	}
	// }
	return []string{}
}

//--------------------------------------------------[ GET SPECIAL PROPERTIES ]--

// Dock icon types.
const (
	IconTypeApplet    = "Applet"
	IconTypeLauncher  = "Launcher"
	IconTypeSeparator = "Separator"
	IconTypeSubDock   = "Stack-icon"
)

// CDIcon defines a dock icon properties.
//
type CDIcon struct {
	// DisplayedName string      // name of the package

	Name     string
	Xid      uint32
	Position int32
	Type     string // Applet, Launcher, Stack-icon, Separator
	// TODO compare
	// Type          PackageType // type of package : installed, user, distant...
	QuickInfo  string
	Container  string
	Command    string
	Order      float64
	ConfigFile string
	Icon       string
	Class      string
	Module     string
}

// FormatName return the user readable name for the applet.
//
func (icon *CDIcon) FormatName() (name string) {
	switch icon.Type {
	case IconTypeApplet:
		name = icon.Module
	case IconTypeSeparator:
		name = "--------"
	default:
		name = icon.Name
		// log.DEV(name, icon)
	}
	return
}

// ListIcons asks the dock the list of active icons.
//
// TODO: add argument for advanced queries.
// would be cool to have argument list.
//
func ListIcons() (list []*CDIcon) {
	for _, props := range DockProperties("type=Icon") {
		pack := &CDIcon{}
		for k, v := range props {
			switch k {
			case "name":
				pack.Name = v.Value().(string)
			case "Xid":
				pack.Xid = v.Value().(uint32)
			case "position":
				pack.Position = v.Value().(int32)
			case "type":
				pack.Type = v.Value().(string)
			case "quick-info":
				pack.QuickInfo = v.Value().(string)
			case "container":
				pack.Container = v.Value().(string)
			case "command":
				pack.Command = v.Value().(string)
			case "order":
				pack.Order = v.Value().(float64)
			case "config-file":
				pack.ConfigFile = v.Value().(string)
			case "icon":
				pack.Icon = v.Value().(string)
			case "class":
				pack.Class = v.Value().(string)
			case "module":
				pack.Module = v.Value().(string)
			default:
				log.Info("ListIcons key not found: "+k, v)
			}
		}
		// if pack.Name == "" {
		// 	log.DEV("*****NONAME", pack.Type, pack.ConfigFile)
		// 	// } else {
		// 	// 	log.DEV(pack.Name, pack.Order)
		// }
		list = append(list, pack)
	}
	return
}

// func ListLaunchers() (list []*CDIcon) {
// 	for _, props := range DockProperties("type=Launcher") {
// 		pack := &CDIcon{}
// 		for k, v := range props {
// 			switch k {
// 			case "name":
// 				pack.Name = v.Value().(string)
// 			case "Xid":
// 				pack.Xid = v.Value().(uint32)
// 			case "position":
// 				pack.Position = v.Value().(int32)
// 			case "type":
// 				pack.Type = v.Value().(string)
// 			case "quick-info":
// 				pack.QuickInfo = v.Value().(string)
// 			case "container":
// 				pack.Container = v.Value().(string)
// 			case "command":
// 				pack.Command = v.Value().(string)
// 			case "order":
// 				pack.Order = v.Value().(float64)
// 			case "config-file":
// 				pack.ConfigFile = v.Value().(string)
// 			case "icon":
// 				pack.Icon = v.Value().(string)
// 			case "class":
// 				pack.Class = v.Value().(string)
// 			case "module":
// 				pack.Module = v.Value().(string)
// 			default:
// 				log.Info("ListIcons key not found: "+k, v)
// 			}
// 		}
// 		// if pack.Name == "" {
// 		// 	log.DEV("*****NONAME", pack.Type, pack.ConfigFile)
// 		// 	// } else {
// 		// 	// 	log.DEV(pack.Name, pack.Order)
// 		// }
// 		list = append(list, pack)
// 	}
// 	return
// }

// InfoApplet asks the dock all informations about an applet.
//
func InfoApplet(name string) *packages.AppletPackage {
	for _, props := range DockProperties("type=Module & name=" + name) {
		return parseApplet(props)
	}
	return nil
}

// ListApplets asks the dock informations about all known applets.
//
func ListApplets() map[string]*packages.AppletPackage {
	list := make(map[string]*packages.AppletPackage)
	// var list []*packages.AppletPackage
	for _, props := range DockProperties("type=Module") {
		pack := parseApplet(props)

		// println(pack.DisplayedName)

		if pack.DisplayedName != "" {
			list[pack.DisplayedName] = pack
			// list = append(list, pack)
			// log.Info("----------------")
			// log.DETAIL(pack)
		}
	}
	return list
}

func parseApplet(props map[string]dbus.Variant) *packages.AppletPackage {
	pack := &packages.AppletPackage{}
	for k, v := range props {
		switch k {
		case "type": // == "Module"

		case "name":
			pack.DisplayedName = v.Value().(string)

		case "title":
			pack.Title = v.Value().(string)

		case "author":
			pack.Author = v.Value().(string)

		case "instances":
			if instances, ok := v.Value().([]string); ok {
				pack.Instances = instances
			}

		case "icon":
			pack.Icon = v.Value().(string)

		case "description":
			pack.Description = v.Value().(string)

		case "is-multi-instance":
			pack.IsMultiInstance = v.Value().(bool)

		case "category":
			if cat, ok := v.Value().(uint32); ok {
				pack.Category = int(cat)
			}

		case "preview":
			pack.Preview = v.Value().(string)

		case "module-type":
			pack.ModuleType = int(v.Value().(uint32))

		default:
			log.Info("ListApplets unmatched", k, v)
		}
	}
	return pack
}

// type AppletPackage struct {
// 	DisplayedName string      // name of the package
// 	Type          PackageType // type of package : installed, user, distant...
// 	Path          string      // complete path of the package.
// 	LastModifDate string      `conf:"last modif"` // date of latest changes in the package.

// 	// On server only.
// 	CreationDate int     `conf:"creation"` // date of creation of the package.
// 	Size         float64 `conf:"size"`     // size in Mo
// 	// Rating int

// 	Author      string `conf:"author"` // author(s)
// 	Description string `conf:"description"`
// 	Category    int    `conf:"category"`

// 	Version       string `conf:"version"`
// 	ActAsLauncher bool   `conf:"act as launcher"`

// }
