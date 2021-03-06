// Package confcore provides a cairo-dock core configuration widget.
package confcore

import (
	"github.com/gotk3/gotk3/gtk"

	"github.com/sqp/godock/libs/cdtype"       // Logger type.
	"github.com/sqp/godock/libs/dock/confown" // New dock own settings.
	"github.com/sqp/godock/libs/text/tran"    // Translate.

	"github.com/sqp/godock/widgets/appinfo"
	"github.com/sqp/godock/widgets/cfbuild"
	"github.com/sqp/godock/widgets/cfbuild/cftype"
	"github.com/sqp/godock/widgets/cfbuild/datatype"
	"github.com/sqp/godock/widgets/common"
	"github.com/sqp/godock/widgets/confapplets"
	"github.com/sqp/godock/widgets/confgui/btnaction"
	"github.com/sqp/godock/widgets/confshortkeys"
	"github.com/sqp/godock/widgets/devpage"
	"github.com/sqp/godock/widgets/docktheme"
	"github.com/sqp/godock/widgets/gtk/buildhelp"
	"github.com/sqp/godock/widgets/gtk/gunvalue"
	"github.com/sqp/godock/widgets/gtk/newgtk"
	"github.com/sqp/godock/widgets/helpfile"
	"github.com/sqp/godock/widgets/pageswitch"
	"github.com/sqp/godock/widgets/report"
	"github.com/sqp/godock/widgets/welcome"

	"errors"
	"path/filepath"
)

const iconSize = 24
const panedPosition = 200

// Custom config tabs displayed with the core page.
const (
	TabDownload  = "Download"  // Key for the download tab.
	TabShortkeys = "Shortkeys" // ...
	TabThemes    = "Themes"
	TabHelp      = "Help"
	TabReport    = "Report"
	TabDev       = "Dev"
)

//
//-----------------------------------------------------------[ CONFCORE DATA ]--

// Item defines a core config page information.
//
type Item struct {
	Key      string
	Title    string
	Icon     string
	Tooltip  string
	Managers []string
	// File string
	// Group string // group in file.
}

var coreItems = []*Item{
	{
		Key:      "Position",
		Title:    tran.Slate("Position"),
		Icon:     "icons/icon-position.svg",
		Tooltip:  tran.Slate("Set the position of the main dock."),
		Managers: []string{"Docks"},
	}, {
		Key:      "Accessibility",
		Title:    tran.Slate("Visibility"),
		Icon:     "icons/icon-visibility.svg",
		Tooltip:  tran.Slate("Do you like your dock to be always visible,\n or on the contrary unobtrusive?\nConfigure the way you access your docks and sub-docks!"),
		Managers: []string{"Docks"},
	}, {
		Key:      "TaskBar",
		Title:    tran.Slate("TaskBar"),
		Icon:     "icons/icon-taskbar.png",
		Tooltip:  tran.Slate("Display and interact with currently open windows."),
		Managers: []string{"Taskbar"},
	}, {
		Key:      "Style",
		Title:    tran.Slate("Style"),
		Icon:     "icons/icon-style.svg",
		Tooltip:  tran.Slate("Configure the global style."),
		Managers: []string{"Style"},
	}, {
		Key:      "Background",
		Title:    tran.Slate("Background"),
		Icon:     "icons/icon-background.svg",
		Tooltip:  tran.Slate("Configure docks appearance."),
		Managers: []string{"Backends", "Docks"}, // -> "dock rendering"

	}, {
		Key:      "Views",
		Title:    tran.Slate("Views"),
		Icon:     "icons/icon-views.svg",
		Tooltip:  tran.Slate("Configure docks appearance."), // same as background (were grouped).
		Managers: []string{"Backends", "Docks"},             // -> "dock rendering"

	}, {
		Key:      "Dialogs",
		Title:    tran.Slate("Dialog boxes and Menus"),
		Icon:     "icons/icon-dialogs.svg",
		Tooltip:  tran.Slate("Configure text bubble appearance."),
		Managers: []string{"Dialogs"}, // -> "dialog rendering"

	}, {
		Key:      "Desklets",
		Title:    tran.Slate("Desklets"),
		Icon:     "icons/icon-desklets.svg",
		Tooltip:  tran.Slate("Applets can be displayed on your desktop as widgets."),
		Managers: []string{"Desklets"}, // -> "desklet rendering"

	}, {
		Key:      "Icons",
		Title:    tran.Slate("Icons"),
		Icon:     "icons/icon-icons.svg",
		Tooltip:  tran.Slate("All about icons:\n size, reflection, icon theme,..."),
		Managers: []string{"Icons", "Indicators"}, // indicators needed here too ?

	}, {
		Key:      "Labels",
		Title:    tran.Slate("Captions"),
		Icon:     "icons/icon-labels.svg",
		Tooltip:  tran.Slate("Define icon caption and quick-info style."),
		Managers: []string{"Icons"},
	}, {
		Key:     TabThemes,
		Title:   tran.Slate("Themes"),
		Icon:    "icons/icon-appearance.svg", // icon-controler.svg
		Tooltip: tran.Slate("Try new themes and save your theme."),
	}, {
		Key:     TabShortkeys,
		Title:   tran.Slate("Shortkeys"),
		Icon:    "icons/icon-shortkeys.svg",
		Tooltip: tran.Slate("Define all the keyboard shortcuts currently available."),
	}, {
		Key:      "System",
		Title:    tran.Slate("System"),
		Icon:     "icons/icon-system.svg",
		Tooltip:  tran.Slate("All of the parameters you will never want to tweak."),
		Managers: []string{"Backends", "Containers", "Connection", "Docks"},
	}, {
		Key:     TabDownload,
		Title:   tran.Slate("Download"),
		Icon:    "icons/retach-desklet.svg",
		Tooltip: tran.Slate("Download additional applets."),
		// "Third-party applets provide integration with many programs, like Pidgin"
	}, {
		Key:     confown.GuiGroup, // custom page for the config own settings.
		Title:   tran.Slate("New Settings"),
		Icon:    "icons/icon-movment.png",
		Tooltip: tran.Slate("New config options, things improved since the original dock."),
	}, {
		Key:     TabReport,
		Title:   tran.Slate("Report"),
		Icon:    "plug-ins/System-monitor/icon.png",
		Tooltip: tran.Slate("Program and system informations for reports and monitoring."),
	}, {
		Key:     TabDev,
		Title:   tran.Slate("Dev"),
		Icon:    "icons/icon-controler.svg",
		Tooltip: tran.Slate("Developer tools."),
	}, {
		Key:     "appInfo",
		Title:   tran.Slate("Edit applet info"),
		Icon:    "cairo-dock.svg",
		Tooltip: tran.Slate("Edit applet registration informations like version or description."),
	}, {
		Key:     "Help",
		Title:   tran.Slate("Help"),
		Icon:    "plug-ins/Help/icon.svg",
		Tooltip: tran.Slate("There are no problems, only solutions (and a lot of useful hints!)"),
	},

	// + icon effects*
	// _add_sub_group_to_group_button (pGroupDescription, "Indicators", "icon-indicators.svg", _("Indicators"));

}

//
//--------------------------------------------------------[ PAGE GUI ICONS ]--

// configWidget extends the Widget interface with common needed actions.
//
type configWidget interface {
	gtk.IWidget
	Destroy()
	ShowAll()
}

type grabber interface {
	configWidget
	Load()
	Grab()
}

type saver interface {
	configWidget
	Save()
}

// ConfCore provides a configuration widget for the main cairo-dock config.
//
type ConfCore struct {
	gtk.Paned

	list     *List
	config   configWidget
	switcher *pageswitch.Switcher
	btn      btnaction.Tune

	data cftype.Source
	log  cdtype.Logger
}

// New creates a ConfCore widget to edit the main cairo-dock config.
//
func New(data cftype.Source, log cdtype.Logger, switcher *pageswitch.Switcher, btn btnaction.Tune) *ConfCore {
	paned := newgtk.Paned(gtk.ORIENTATION_HORIZONTAL)

	widget := &ConfCore{
		Paned:    *paned,
		switcher: switcher,
		btn:      btn,
		data:     data,
		log:      log,
	}
	widget.list = NewList(widget, log)
	widget.Pack1(widget.list, true, true)

	widget.SetPosition(panedPosition)
	return widget
}

// Load loads config items in the widget.
//
func (widget *ConfCore) Load() {
	widget.list.Load(widget.data.DirShareData())
}

// Selected returns the selected page config.
//
// func (widget *ConfCore) Selected() (*Item, error) {
// 	return widget.list.Selected()
// }

// func (widget *ConfCore) Clean() {
// }

// Select sets the selected item based on its key name.
//
func (widget *ConfCore) Select(key string) bool {
	return widget.list.Select(key)
}

// ShowWelcome shows the welcome placeholder widget if nothing is displayed.
//
func (widget *ConfCore) ShowWelcome(setBtn bool) {
	if widget.config == nil {
		widget.setCurrent(welcome.New(widget.data, widget.log))
		if setBtn {
			widget.btn.SetNone()
		}
	}
}

//--------------------------------------------------------[ SAVE CONFIG PAGE ]--

// Save saves the current page configuration
//
func (widget *ConfCore) Save() {
	if widget.config == nil {
		return
	}

	if grab := widget.grabber(); grab != nil {
		grab.Grab()
		return
	}

	if saver, ok := widget.config.(saver); ok {
		saver.Save()
	}
	// //\_____________ reload modules that are concerned by these changes
	// GldiManager *pManager;
	// if (bUpdateColors)
	// {
	// 	cd_reload ("Style");
	// 	cd_reload ("Indicators");
	// 	cd_reload ("Dialogs");
	// 	GldiModule *pModule;
	// 	pModule = gldi_module_get ("clock");
	// 	if (pModule)
	// 		gldi_object_reload (GLDI_OBJECT (pModule), TRUE);
	// 	pModule = gldi_module_get ("keyboard indicator");
	// 	if (pModule)
	// 		gldi_object_reload (GLDI_OBJECT (pModule), TRUE);
	// 	pModule = gldi_module_get ("dock rendering");
	// 	if (pModule)
	// 		gldi_object_reload (GLDI_OBJECT (pModule), TRUE);
	// }
	// cd_reload ("Docks");
	// cd_reload ("Taskbar");
	// cd_reload ("Icons");
	// cd_reload ("Backends");

	// if (pModuleInstanceAnim != NULL)
	// {
	// 	gldi_object_reload (GLDI_OBJECT(pModuleInstanceAnim), TRUE);
	// }
	// if (pModuleInstanceEffect != NULL)
	// {
	// 	gldi_object_reload (GLDI_OBJECT(pModuleInstanceEffect), TRUE);
	// }
	// if (pModuleInstanceIllusion != NULL)
	// {
	// 	gldi_object_reload (GLDI_OBJECT(pModuleInstanceIllusion), TRUE);
	// }
}

func (widget *ConfCore) grabber() grabber {
	if widget.config != nil {
		grab, ok := widget.config.(grabber)
		if ok {
			return grab
		}
	}
	return nil
}

//-------------------------------------------------------[ CONTROL CALLBACKS ]--

// onSelect acts as a row selected callback.
//
func (widget *ConfCore) onSelect(item *Item, e error) {
	if widget.log.Err(e, "onSelect: selection problem") {
		return
	}
	widget.switcher.Clear()

	if widget.config != nil {
		widget.config.Destroy()
		widget.config = nil
	}

	var w configWidget
	switch item.Key {

	// Custom widgets.

	case "appInfo":
		w = appinfo.NewLoaded(widget.data, widget.log, widget.switcher)
		widget.btn.SetSave()

	case TabShortkeys:
		w = confshortkeys.New(widget.data, widget.log, widget.btn)
		widget.btn.SetGrab()

	case TabThemes:
		var ok bool
		w, ok = docktheme.New(widget.data, widget.log, widget.switcher)
		if ok {
			widget.btn.SetApply()
		} else {
			widget.btn.SetNone()
		}

	case TabDownload:
		w = confapplets.NewLoaded(widget.data, widget.log, nil, confapplets.ListExternal)
		widget.btn.SetNone()

	case TabHelp:
		w, _ = helpfile.New(widget.data, widget.log, widget.switcher)
		widget.btn.SetNone()

	case TabReport:
		w = report.New(widget.data, widget.log, widget.switcher)
		widget.btn.SetNone()

	case TabDev:
		w = devpage.New(widget.data, widget.log, widget.switcher)
		widget.btn.SetNone()

		// Custom file path.

	case confown.GuiGroup: // own config has a special path.
		w = widget.fromFile(item,
			confown.PathFile(),
			"", // no default.
			func() { confown.Current.Load() }, // reload own conf if saved.
		)

		// Default file path.

	default:
		w = widget.fromFile(item,
			widget.data.MainConfigFile(),
			widget.data.MainConfigDefault(),
			func() {
				tokf, ok := interface{}(w).(cfbuild.KeyFiler)
				if !ok {
					widget.log.NewErr("bad config widget", "update confcore")
					return
				}
				kf := tokf.KeyFile()
				if kf == nil {
					conf, e := cfbuild.LoadFile(widget.data.MainConfigFile(), "")
					if widget.log.Err(e, "update confcore, reload conf file") {
						return
					}
					kf = &conf.KeyFile
					defer kf.Free()
				}

				for _, manager := range item.Managers {
					widget.data.ManagerReload(manager, true, kf)
				}
			})
	}

	widget.setCurrent(w)
}

func (widget *ConfCore) fromFile(item *Item, file, defFile string, postSave func()) configWidget {
	build, ok := cfbuild.NewFromFileSafe(widget.data, widget.log, file, defFile, "")

	if ok {
		build.BuildSingle(item.Key)
		build.SetPostSave(postSave)
		widget.btn.SetSave()

	} else {
		widget.btn.SetNone()
	}
	return build
}

func (widget *ConfCore) setCurrent(w configWidget) {
	widget.config = w
	widget.Pack2(w, true, true)
}

// UpdateModuleState updates the state of the given applet, from a dock event.
//
func (widget *ConfCore) UpdateModuleState(name string, active bool) {
	if widget.config == nil {
		return
	}
	confapp, ok := widget.config.(datatype.UpdateModuleStater)
	if !ok {
		return
	}

	confapp.UpdateModuleState(name, active)
}

// UpdateShortkeys updates the shortkey widget if it's loaded.
//
func (widget *ConfCore) UpdateShortkeys() {
	widget.log.Info("UpdateShortkeys")
	if grab := widget.grabber(); grab != nil {
		grab.Load()
	}
}

//-------------------------------------------------------[ WIDGET CORE LIST ]--

// Liststore rows. Must match the ListStore declaration type and order.
const (
	rowKey = iota
	rowIcon
	rowName
	rowTooltip
)

// controlItems forwards events to other widgets.
//
type controlItems interface {
	onSelect(*Item, error)
}

// row defines a pointer to link the icon name with its iter.
//
type row struct {
	Iter *gtk.TreeIter
	Conf *Item
}

// List is a widget to list and select cairo-dock main config pages references.
//
type List struct {
	gtk.ScrolledWindow // Main widget is the container. The ScrolledWindow will handle list scrollbars.
	tree               *gtk.TreeView
	model              *gtk.ListStore
	selection          *gtk.TreeSelection
	control            controlItems
	log                cdtype.Logger

	rows map[string]*row
}

// NewList creates cairo-dock main config pages list.
//
func NewList(control controlItems, log cdtype.Logger) *List {
	builder := buildhelp.NewFromBytes(confcoreXML())

	widget := &List{
		ScrolledWindow: *builder.GetScrolledWindow("widget"),
		model:          builder.GetListStore("model"),
		tree:           builder.GetTreeView("tree"),
		selection:      builder.GetTreeSelection("selection"),
		control:        control,
		log:            log,
		rows:           make(map[string]*row),
	}

	if len(builder.Errors) > 0 {
		for _, e := range builder.Errors {
			log.Err(e, "build confcore list")
		}
		return nil
	}

	// Action: Treeview Select line.
	widget.selection.Connect("changed", widget.onSelectionChanged)

	return widget
}

// Load loads the widget fields.
//
func (widget *List) Load(shareData string) {
	for _, item := range coreItems {
		iter := widget.model.Append()
		widget.rows[item.Key] = &row{iter, item}

		args := gtk.Cols{
			rowKey:     item.Key,
			rowName:    item.Title,
			rowTooltip: item.Tooltip,
		}
		widget.model.SetCols(iter, args)

		if item.Icon != "" {
			path := filepath.Join(shareData, item.Icon)
			if pix, e := common.PixbufNewFromFile(path, iconSize); !widget.log.Err(e, "Load icon") {
				widget.model.SetValue(iter, rowIcon, pix)
			}
		}
	}
}

// Select sets the selected row based on its name.
//
func (widget *List) Select(conf string) bool {
	row, ok := widget.rows[conf]
	if !ok {
		return false
	}

	widget.selection.SelectIter(row.Iter)
	return true
}

// Selected returns the data about the selected item.
//
func (widget *List) Selected() (*Item, error) {
	key, e := gunvalue.SelectedValue(widget.model, widget.selection, rowKey).String()
	if e != nil {
		return nil, e
	}
	conf, ok := widget.rows[key]
	if !ok {
		// TODO:  warn
		return nil, errors.New("no matching row found")
	}
	return conf.Conf, nil
}

//-------------------------------------------------------[ ACTIONS CALLBACKS ]--

// Selected line has changed. Forward the call to the controler.
//
func (widget *List) onSelectionChanged(obj *gtk.TreeSelection) {
	widget.control.onSelect(widget.Selected())
}

//

/*
		_("Icons"));
	_add_sub_group_to_group_button (pGroupDescription, "Icons", "icon-icons.svg", _("Icons"));
	_add_sub_group_to_group_button (pGroupDescription, "Indicators", "icon-indicators.svg", _("Indicators"));
	pGroupDescription->pManagers = g_list_prepend (pGroupDescription->pManagers, (gchar*)"Icons");
	pGroupDescription->pManagers = g_list_prepend (pGroupDescription->pManagers, (gchar*)"Indicators");  // -> "drop indicator"

*/
