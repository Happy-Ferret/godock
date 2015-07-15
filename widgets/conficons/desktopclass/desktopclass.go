// Package desktopclass provides a desktop class informations widget.
package desktopclass

import (
	"github.com/conformal/gotk3/gtk"

	"github.com/sqp/godock/libs/text/strhelp"

	"github.com/sqp/godock/widgets/common"
	"github.com/sqp/godock/widgets/confbuilder/datatype"

	"os"
	"path/filepath"
	"strings"
)

// New creates a desktop class informations widget.
//
func New(source datatype.Source, selected datatype.DesktopClasser, origins string) gtk.IWidget {
	apps := strings.Split(origins, ";")
	if len(apps) == 0 {
		return nil
	}

	// Remove the path from the first item.
	dir := filepath.Dir(apps[0])
	apps[0] = filepath.Base(apps[0])

	// Try force select the first one (can be inactive if "do not bind appli").
	if selected.String() == "" {
		selected = source.DesktopClasser(strings.TrimSuffix(apps[0], ".desktop"))
	}

	command := selected.Command()
	desktopFile := desktopFileText(apps, dir, selected.String())

	wName := boxLabel(selected.Name())
	wIcon := boxLabel(selected.Icon())
	wCommand := boxButton(command, func() { println("need to launch", command) })
	wDesktopFiles := boxLabel(desktopFile)

	grid, _ := gtk.GridNew()
	grid.Attach(boxLabel("Name"), 0, 0, 1, 1)
	grid.Attach(boxLabel("Icon"), 0, 1, 1, 1)
	grid.Attach(boxLabel("Command"), 0, 2, 1, 1)
	grid.Attach(boxLabel("Desktop file"), 0, 3, 1, 1)
	grid.Attach(wName, 1, 0, 1, 1)
	grid.Attach(wIcon, 1, 1, 1, 1)
	grid.Attach(wCommand, 1, 2, 1, 1)
	grid.Attach(wDesktopFiles, 1, 3, 1, 1)

	frame, _ := gtk.FrameNew("")
	label, _ := gtk.LabelNew(common.Bold("Launcher origin"))
	label.SetUseMarkup(true)
	frame.SetLabelWidget(label)
	frame.Add(grid)
	return frame
}

func boxButton(label string, call func()) *gtk.Box {
	btnCommand, _ := gtk.ButtonNewWithLabel(label)
	btnCommand.Connect("clicked", call)
	btnCommand.SetRelief(gtk.RELIEF_HALF)
	return boxWidget(btnCommand)
}

func boxLabel(str string) *gtk.Box {
	label, _ := gtk.LabelNew(str + "\t")
	label.SetUseMarkup(true)
	return boxWidget(label)
}

func boxWidget(widget gtk.IWidget) *gtk.Box {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(widget, false, false, 0)
	return box
}

func fileExists(path string) bool {
	_, e := os.Stat(path)
	return e == nil || !os.IsNotExist(e)

}

func desktopFileText(apps []string, dir, selected string) string {
	text := ""
	for _, v := range apps {
		// Remove suffix for name and highlight the active one (with link if possible).
		name := strings.TrimSuffix(v, ".desktop")
		isCurrent := name == selected

		if fileExists(filepath.Join(dir, v)) {
			name = common.URI("file://"+filepath.Join(dir, v), name)
		}

		if isCurrent {
			name = common.Bold(name)
		}
		text = strhelp.Separator(text, ", ", name)
	}
	return text
}
