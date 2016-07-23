// Package cfbuild builds a cairo-dock configuration widget from its config file.
package cfbuild

import (
	"github.com/sqp/godock/libs/cdtype" // Logger type.

	"github.com/sqp/godock/widgets/cfbuild/cftype"   // Types for config file builder usage.
	"github.com/sqp/godock/widgets/cfbuild/newkey"   // Create config file builder keys.
	"github.com/sqp/godock/widgets/cfbuild/valuer"   // Converts interface value.
	"github.com/sqp/godock/widgets/cfbuild/vstorage" // virtual config storage.
	"github.com/sqp/godock/widgets/gtk/keyfile"      // real storage with the C keyfile (same as the dock for 100% compatibility).
	"github.com/sqp/godock/widgets/pageswitch"       // page switcher for multi groups.

	"errors"
	"strconv"
	"strings"
)

//
//-----------------------------------------------------------------[ GROUPER ]--

// Grouper builds config pages from the Builder.
//
type grouper struct {
	cftype.Builder
	free    func()
	keyFile *keyfile.KeyFile
}

// newFromStorage creates a config page builder with the given config storage.
//
func newFromStorage(source cftype.Source, log cdtype.Logger, storage cftype.Storage, originalConf, gettextDomain string) *grouper {
	grouper := &grouper{
		Builder: NewBuilder(source, log, storage, originalConf, gettextDomain),
	}
	storage.SetBuilder(grouper)
	grouper.free = grouper.Builder.Free
	grouper.Builder.Connect("destroy", func() { grouper.free() })
	return grouper
}

// NewFromFile creates a config page builder from the file.
//
func NewFromFile(source cftype.Source, log cdtype.Logger, configFile, originalConf, gettextDomain string) (cftype.Grouper, error) {
	storage, e := LoadFile(configFile, originalConf)
	if e != nil {
		log.Err(e, "NewFromFile")
		return nil, e
	}
	grouper := newFromStorage(source, log, storage, originalConf, gettextDomain)
	grouper.free = func() {
		grouper.keyFile = &storage.KeyFile // MOVE ONE LINE UP ?
		grouper.Builder.Free()
		storage.KeyFile.Free()
		if storage.def != nil {
			storage.def.Free()
		}
	}
	return grouper, nil
}

// NewFromFileSafe creates a config page builder from the file.
//
// If the load file failed, an error widget is returned with false.
//
func NewFromFileSafe(source cftype.Source, log cdtype.Logger, file, originalConf, gettextDomain string) (cftype.Grouper, bool) {
	build, e := NewFromFile(source, log, file, originalConf, gettextDomain)
	if !log.Err(e, "Load config file", file) {
		return build, true // Load ok, return it.
	}

	// Load failed. Build a warning widget from virtual source.

	conf := vstorage.NewVirtual(file, originalConf)
	build = newFromStorage(source, log, conf, originalConf, gettextDomain)

	group := "problem"
	build.AddGroup(group,
		newkey.Frame(group, "fail", "Load failed", "dialog-error"),
		newkey.TextLabel(group, "text", "Can't load the configuration file to build the interface."),
		newkey.TextLabel(group, "file", file),
	)

	hack := TweakKeyMakeWidget(group, "file", func(key *cftype.Key) {
		key.Label().SetLineWrap(true)
		// key.Label().SetJustify(gtk.JUSTIFY_FILL)
		key.Label().SetSelectable(true)
	})

	return build.BuildSingle(group, hack), false
}

// NewVirtual creates a config page builder with an empty virtual storage.
//
// configFile is unused. Can be assigned to anything (and reused for the save).
//
func NewVirtual(source cftype.Source, log cdtype.Logger, configFile, originalConf, gettextDomain string) cftype.Grouper {
	conf := vstorage.NewVirtual(configFile, originalConf)
	return newFromStorage(source, log, conf, originalConf, gettextDomain)
}

//
//-------------------------------------------------------------------[ BUILD ]--

// BuildSingle builds a single page config for the given group.
//
func (build *grouper) BuildSingle(group string, tweaks ...func(cftype.Builder)) cftype.Grouper {
	// Add keys for the group.
	build.AddGroup(group, build.Storage().List(group)...)

	// Apply tweaks.
	for _, tw := range tweaks {
		tw(build)
	}

	w := build.BuildPage(group)
	build.PackStart(w, true, true, 0)
	build.ShowAll()
	return build
}

// BuildAll builds a dock configuration widget with all groups.
//
func (build *grouper) BuildAll(switcher *pageswitch.Switcher, tweaks ...func(cftype.Builder)) cftype.Grouper {
	_, groups := build.Storage().GetGroups()
	return build.BuildGroups(switcher, groups, tweaks...)
}

// BuildGroups builds a dock configuration widget with the given groups.
//
func (build *grouper) BuildGroups(switcher *pageswitch.Switcher, groups []string, tweaks ...func(cftype.Builder)) cftype.Grouper {
	// Load keys.
	for _, group := range groups {
		keys := build.Storage().List(group)
		build.AddGroup(group, keys...)
	}

	// Apply tweaks.
	for _, tw := range tweaks {
		tw(build)
	}

	// Build groups.
	first := true
	for _, group := range build.Groups() {
		w := build.BuildPage(group)
		switcher.AddPage(&pageswitch.Page{
			Key:     group,
			Name:    build.Translate(group),
			OnShow:  func() { build.PackStart(w, true, true, 0); w.ShowAll() },
			OnHide:  func() { build.Remove(w) },
			OnClear: w.Destroy})

		if first {
			switcher.Activate(group)
			first = false
		}
	}

	// Single group, hide the switcher. Multi groups, display it.
	switcher.SetVisible(len(groups) > 1)

	build.ShowAll()
	return build
}

// KeyFiler defines the interface to recognise a grouper (provides its KeyFile).
//
type KeyFiler interface {
	KeyFile() *keyfile.KeyFile
}

// KeyFile returns the pointer to the internal KeyFile.
//
func (build *grouper) KeyFile() *keyfile.KeyFile {
	return build.keyFile
}

//
//-----------------------------------------------------------[ CONFIG SOURCE ]--

// CDConfig loads data from a Cairo-Dock configuration file. Implements cftype.Storage.
//
type CDConfig struct {
	keyfile.KeyFile
	cftype.BaseStorage // filepath and build.

	def *keyfile.KeyFile // default values.
}

// LoadFile loads a Cairo-Dock configuration file as *CDConfig.
func LoadFile(configFile, configDefault string) (*CDConfig, error) {
	pKeyF, e := keyfile.NewFromFile(configFile, keyfile.FlagsKeepComments|keyfile.FlagsKeepTranslations)
	if e != nil {
		return nil, e
	}
	return &CDConfig{
		KeyFile: *pKeyF,
		BaseStorage: cftype.BaseStorage{
			File:    configFile,
			Default: configDefault,
		},
	}, nil
}

// List lists keys defined in the configuration file.
//
func (conf *CDConfig) List(group string) (list []*cftype.Key) {
	_, keys, _ := conf.GetKeys(group) // (uint64, []string, error)
	for _, cKeyName := range keys {
		comment, _ := conf.GetComment(group, cKeyName)

		if key := conf.ParseKeyComment(group, cKeyName, comment); key != nil {
			list = append(list, key)
		}
	}
	return
}

// Valuer gives access to a field value.
//
func (conf *CDConfig) Valuer(group, name string) valuer.Valuer {
	return keyfile.NewValuer(&conf.KeyFile, group, name)
}

// Default gives access to a field value.
//
func (conf *CDConfig) Default(group, name string) (valuer.Valuer, error) {
	e := conf.loadDefault()
	if e != nil {
		return nil, e
	}

	return keyfile.NewValuer(conf.def, group, name), nil
}

// loadDefault gives access to a field value.
//
func (conf *CDConfig) loadDefault() error {
	if conf.def != nil { // already loaded.
		return nil
	}
	fileDefault := conf.FileDefault()
	if fileDefault == "" { // no path
		return errors.New("no default file path")
	}
	kf, e := keyfile.NewFromFile(fileDefault, keyfile.FlagsKeepComments|keyfile.FlagsKeepTranslations)
	if e != nil {
		return e
	}

	conf.def = kf
	return nil
}

//
//-----------------------------------------------------------------[ PARSING ]--

// ParseKeyComment parse comments for a key.
//
func (conf *CDConfig) ParseKeyComment(group, name, cKeyComment string) *cftype.Key {
	cUsefulComment := strings.TrimLeft(cKeyComment, "# \n")  // remove #, spaces, and endline from start.
	cUsefulComment = strings.TrimRight(cUsefulComment, "\n") // remove endline from end.

	// Drop invalid or too short comments.
	if len(cKeyComment) < 2 || len(cUsefulComment) == 0 || cUsefulComment[0] == '[' {
		// '[' : on gere le bug de la Glib, qui rajoute les nouvelles cles apres le commentaire du groupe suivant !
		// log.DEV("LIBC BUG, DETECTED", comment) // often seem to be a [gtk-convert]
		return nil
	}

	key := cftype.KeyType(cUsefulComment[0]).New(conf.Build, group, name)

	cUsefulComment = cUsefulComment[1:]

	for i, c := range cUsefulComment {
		if c != '-' && c != '+' && c != ' ' {
			if c == cftype.FlagCairoOnly {
				// If opengl, need drop key
				// datatype.DisplayModeOpenGL

			} else if c == cftype.FlagOpenGLOnly {
				// If !opengl, need drop key
				//  datatype.DisplayModeCairo

			} else {
				// Try to detect a value indicating the number of elements.
				key.NbElements, _ = strconv.Atoi(string(cUsefulComment[i:]))

				// Try to get authorized values between square brackets.
				if c == '[' {
					values := cUsefulComment[i+1 : strings.Index(cUsefulComment, "]")]
					i += len(values) + 1

					key.AuthorizedValues = strings.Split(values, ";")
				}

				// End of arguments at the start .
				cUsefulComment = cUsefulComment[i:]
				break
			}
		}
	}

	if key.NbElements == 0 {
		key.NbElements = 1
	}

	cUsefulComment = strings.TrimLeft(cUsefulComment, "]1234567890") // Remove last bits of possible arguments.
	cUsefulComment = strings.TrimLeft(cUsefulComment, " ")           // Remove separator.

	// log.DEV("parsed", string(iType), iNbElements, cUsefulComment)

	// Special widget alignment with a trailing slash.
	if strings.HasSuffix(cUsefulComment, "/") {
		cUsefulComment = strings.TrimSuffix(cUsefulComment, "/")
		key.IsAlignedVertical = true
	}

	// Get tooltip.
	toolStart := strings.IndexByte(cUsefulComment, '{')
	toolEnd := strings.IndexByte(cUsefulComment, '}')
	if toolStart > 0 && toolEnd > 0 && toolStart < toolEnd {
		key.Tooltip = cUsefulComment[toolStart+1 : toolEnd]
		cUsefulComment = cUsefulComment[:toolStart-1]
	}

	key.Text = cUsefulComment

	return key
}
