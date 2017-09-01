/*
Package config is an automatic configuration loader for cairo-dock.

Config will fills the data of a struct from an INI config file with reflection.
Groups and keys in the file can be matched with the data struct by name or by a
special "conf" tag.

	GetKey  : Only parse using the field name. Names and keys need to be UpperCase.
	GetTag  ; Only parse using the "conf" tag of the field.
	GetBoth : Parse using both methods (tag is used when defined, name as fallback).

Parsing errors are stored in the Errors field.

Example for a single group

Load the data from the file and UnmarshalGroup a group.

	conf, e := config.NewFromFile(filepath) // Special conf reflector around the config file parser.
	if e != nil {
		return e
	}
	data := &groupConfiguration{}
	conf.UnmarshalGroup(data, groupName, config.GetKey)

Example with multiple groups

To load data from many groups splitted in according strucs, like applets config,
you have to define the main struct with a "group" tag that match the group in
the config file.

	data := &appletConf{}
	e := config.Load(filepath, data, config.GetBoth)

Structs data for the examples

This is an example of applet data with the common Icon group (Name, Debug, and
optional Icon).

	type appletConf struct {
		cdtype.ConfGroupIconBoth `group:"Icon"`
		groupConfiguration       `group:"Configuration"`
	}

	type groupConfiguration struct {
		DisplayText   int
		DisplayValues int

		GaugeName string

		IconBroken  string
		VolumeStep  int
		StreamIcons bool
	}

*/
package config

import (
	"github.com/go-ini/ini"

	"github.com/sqp/godock/libs/cdglobal"          // Dock types.
	"github.com/sqp/godock/libs/cdtype"            // Applet types.
	"github.com/sqp/godock/libs/files/fileaccess"  // Serialize files access.
	"github.com/sqp/godock/widgets/cfbuild/valuer" // Converts interface value.

	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// ini parser global config.
func init() {
	ini.LineBreak = "\n\n"   // Blank line between fields.
	ini.PrettyFormat = false // Don't try to align equal signs.
}

//
//--------------------------------------------------------------[ MATCH KEYS ]--

// GetKey is a cdtype.GetFieldKey test that matches by the field name.
//
func GetKey(struc reflect.StructField) string {
	return struc.Name
}

// GetTag is a cdtype.GetFieldKey test that matches by the struct tag if defined.
//
func GetTag(struc reflect.StructField) string {
	return struc.Tag.Get("conf")
}

// GetBoth is a cdtype.GetFieldKey test that matches by the struct tag if
//  defined, otherwise use the field name.
//
func GetBoth(struc reflect.StructField) string {
	if tag := struc.Tag.Get("conf"); tag != "" {
		return tag // Got tag, use it.
	}
	return struc.Name // Else, use name.
}

//
//-----------------------------------------------------------------[ LOADING ]--

// Config file unmarshall. Parsing errors will be stacked in the Errors field.
//
type Config struct {
	*ini.File // Extends the real config.

	Errors    []error
	confdir   string             // user config dir.
	appdir    string             // applet dir.
	shortkeys []*cdtype.Shortkey // shortkeys found.
	actions   []func(cdtype.AppAction)

	filePath string      // Full path to config file.
	fileMode os.FileMode // File access rights.
	log      cdtype.Logger
}

// NewEmpty creates a new empty Config parser.
// Also locks files access.
//
func NewEmpty(log cdtype.Logger, configFile string) *Config {
	fileaccess.Lock(log)
	return &Config{
		File:     ini.Empty(),
		filePath: configFile,
		fileMode: 0644,
		log:      log,
	}
}

// NewFromFile creates a ConfUpdater for the given config file (full path).
// This lock files access. Ensure you Save or Cancel fast.
//
func NewFromFile(log cdtype.Logger, configFile string) (*Config, error) {
	// Ensure the file exists and get the file access rights to preserve them.
	var e error
	configFile, e = filepath.Abs(configFile)
	if e != nil {
		return nil, e
	}
	fi, e := os.Stat(configFile)
	if e != nil {
		return nil, e
	}

	log.Debug("lock", configFile)

	fileaccess.Lock(log)

	iniOpts := ini.LoadOptions{IgnoreInlineComment: true} // don't parse # and ; in keys as comments.
	cfg, e := ini.LoadSources(iniOpts, configFile)
	if e != nil {
		fileaccess.Unlock(log)
		return nil, e
	}
	return &Config{
		File:     cfg,
		filePath: configFile,
		fileMode: fi.Mode(),
		log:      log,
	}, nil
}

// NewFromReader creates a new Config parser with reflection to fill fields.
//
func NewFromReader(reader io.Reader) (*Config, error) {
	buf := bytes.NewBuffer(nil)
	io.Copy(buf, reader)
	cfg, e := ini.Load(buf.Bytes())
	if e != nil {
		return nil, e
	}

	return &Config{
		File: cfg,
	}, nil
}

// Load loads a config file and fills a config data struct.
//
// Returns parsed defaults data, the list of parsing errors, and the main error
// if the load failed (file missing / not readable).
//
//   log        Logger.
//   filename   Full path to the config file.
//   appdir     Application directory, to find templates.
//   v          The pointer to the data struct.
//   fieldKey   Func to choose what key to load for each field.
//              Usable methods provided: GetKey, GetTag, GetBoth.
//
func Load(log cdtype.Logger, filename, confdir, appdir string, v interface{}, fieldKey cdtype.GetFieldKey) (cdtype.Defaults, []func(cdtype.AppAction), []error, error) {
	cfg, e := NewFromFile(log, filename)
	if e != nil {
		return cdtype.Defaults{}, nil, nil, e
	}
	cfg.confdir = confdir
	cfg.appdir = appdir
	def := cfg.Unmarshall(v, fieldKey)
	cfg.Cancel()
	return def, cfg.actions, cfg.Errors, nil
}

// SetToFile gets a conf updater in read/write mode.
//
func SetToFile(log cdtype.Logger, filename string, call func(cdtype.ConfUpdater) error) error {
	cfg, e := NewFromFile(log, filename)
	if e != nil {
		return e
	}
	e = call(cfg)
	if e != nil {
		cfg.Cancel()
		return e
	}
	return cfg.Save()
}

// GetFromFile gets a conf updater in read only.
//
func GetFromFile(log cdtype.Logger, filename string, call func(cdtype.ConfUpdater)) error {
	cfg, e := NewFromFile(log, filename)
	if e != nil {
		return e
	}
	call(cfg)
	cfg.Cancel()
	return nil
}

// UpdateFile udates one key in a configuration file.
//
func UpdateFile(log cdtype.Logger, filename, group, key string, value interface{}) error {
	return SetToFile(log, filename, func(cfg cdtype.ConfUpdater) error {
		return cfg.Set(group, key, value)
	})
}

// Cancel releases the file locks.
//
func (c *Config) Cancel() {
	fileaccess.Unlock(c.log)
}

// Save saves the edited config to disk, and releases the file locks.
//
func (c *Config) Save() error {
	defer fileaccess.Unlock(c.log)

	buf := bytes.NewBuffer(nil)
	_, e := c.WriteToIndent(buf, "")
	if e != nil {
		return e
	}

	// Remove empty space at the end, except one endline.
	data := append(bytes.TrimRight(buf.Bytes(), "\n "), []byte("\n")...)
	return ioutil.WriteFile(c.filePath, data, c.fileMode)
}

// Valuer returns the valuer for the given group/key combo.
//
func (c *Config) Valuer(group, key string) valuer.Valuer {
	return &value{
		c:     *c,
		group: group,
		name:  key,
	}
}

// ParseGroups calls the given func for every group with its list of keys.
//
func (c *Config) ParseGroups(call func(group string, keys []cdtype.ConfKeyer)) {
	for _, group := range c.SectionStrings() {
		var keys []cdtype.ConfKeyer
		for _, key := range c.Section(group).Keys() {
			keys = append(keys, &confKey{
				key:    key,
				Valuer: c.Valuer(group, key.Name()),
			})
		}
		call(group, keys)
	}
}

//
//---------------------------------------------------------------[ UNMARSHAL ]--

// Unmarshall fills a struct of struct with data from config.
// The First level is config group, matched by the key group.
// Second level is data fields, matched by the supplied GetFieldKey func.
//
func (c *Config) Unmarshall(v interface{}, fieldKey cdtype.GetFieldKey) cdtype.Defaults {
	typ := reflect.Indirect(reflect.ValueOf(v)).Type().Elem() // Get the type of the struct behind the pointer.
	val := reflect.ValueOf(v).Elem()                          // ReflectValue of the config struct.

	val.Set(reflect.New(typ)) // Create a new empty struct.

	def := cdtype.Defaults{Commands: cdtype.Commands{}} // Empty defaults to gather groups auto set defaults.

	// Range over the first level of fields to find struct with tag "group".
	for i := 0; i < typ.NumField(); i++ { // Parsing all fields in grre.
		// log.Info("field", i, typ.Field(i).Name, typ.Field(i).Tag.Get("group"))
		group := typ.Field(i).Tag.Get("group")
		if group == "" {
			continue
		}
		// Get user data from the group.
		c.unmarshalGroup(val.Elem().Field(i), group, fieldKey)

		// Get applet defaults from the group if it's public and provides a ToDefaults method.
		if val.Elem().Field(i).CanInterface() {
			uncast := val.Elem().Field(i).Interface()
			getDef, ok := uncast.(cdtype.ToDefaultser)
			if ok {
				getDef.ToDefaults(&def)
			}
		}
	}
	def.Shortkeys = c.shortkeys
	return def
}

// UnmarshalGroup parse a config group to fill the ptr to struct provided.
//
// The group param must match a group in the file with the format [MYGROUP]
//
func (c *Config) UnmarshalGroup(v interface{}, group string, fieldKey cdtype.GetFieldKey) []error {
	conf := reflect.ValueOf(v).Elem()
	c.unmarshalGroup(conf, group, fieldKey)
	return c.Errors
}

// see UnmarshalGroup.
func (c *Config) unmarshalGroup(conf reflect.Value, group string, fieldKey cdtype.GetFieldKey) {
	typ := conf.Type()
	for i := 0; i < typ.NumField(); i++ { // Parsing all fields in type.
		elem := conf.Field(i)
		tag := typ.Field(i).Tag
		key := fieldKey(typ.Field(i))
		ck := c.Section(group).Key(key)

		switch {
		case key == "" || key == "-" || !elem.CanInterface(): // Ensure key is valid and data is usable.
			// Dropped.

		case c.fieldFromConfBasic(elem, ck, group, key, tag),
			c.fieldFromConfDock(elem, ck, group, key, tag):
			// Matched.

		case elem.Kind().String() == "int": // Int like types often used as ref types.
			val, e := ck.Int()
			c.testerr(e, group, key, "int like value")
			elem.SetInt(int64(val))

		default:
			c.adderr(group, key, "config.unmarshalGroup unknown type: %T", elem.Interface())
		}
	}
}

// Fill a single reflected field if it has the conf tag.
//
func (c *Config) fieldFromConfBasic(elem reflect.Value, ck *ini.Key, group, key string, tag reflect.StructTag) bool {
	switch elem.Interface().(type) {

	case bool:
		val, e := ck.Bool()
		c.testerr(e, group, key, "bool value")
		elem.SetBool(val)
		e = tagInt(tag.Get("action"), func(id int) {
			// Get the pointer to value for the set value action callback.
			b := elem.Addr().Interface().(*bool)
			c.actions = append(c.actions, func(act cdtype.AppAction) { act.SetBool(id, b) })
		})
		c.testerr(e, group, key, "bool action")

	case int, int32, int64, cdtype.InfoPosition, cdtype.RendererGraphType:
		val, e := ck.Int()
		c.testerr(e, group, key, "int value")
		elem.SetInt(int64(val))

	case string:
		val := ck.String()
		if val == "" {
			val = tag.Get("default")
		}
		elem.SetString(val)

	case []byte:
		val := ck.String()
		if val == "" {
			val = tag.Get("default")
		}
		elem.SetBytes([]byte(val))

	case float64:
		val, e := ck.Float64()
		c.testerr(e, group, key, "float64 value")
		elem.SetFloat(val)

	case []string:
		val := ck.String()
		list := strings.Split(strings.TrimRight(val, ";"), ";")
		if list[len(list)-1] == "" {
			list = list[:len(list)-1]
		}
		elem.Set(reflect.ValueOf(list))

	default:
		return false
	}
	return true
}

func (c *Config) fieldFromConfDock(elem reflect.Value, ck *ini.Key, group, key string, tag reflect.StructTag) bool {
	switch elem.Interface().(type) {

	case cdtype.Duration:
		val, e := ck.Int()
		c.testerr(e, group, key, "Duration value")

		dur := cdtype.NewDuration(val)
		e = dur.SetUnit(tag.Get("unit"))
		c.testerr(e, group, key, "Duration unit")

		e = tagInt(tag.Get("default"), dur.SetDefault)
		c.testerr(e, group, key, "Duration default")

		e = tagInt(tag.Get("min"), dur.SetMin)
		c.testerr(e, group, key, "Duration min")

		elem.Set(reflect.ValueOf(*dur))

	case *cdtype.Shortkey:
		val := ck.String()
		sk := &cdtype.Shortkey{
			ConfGroup: group,
			ConfKey:   key,
			Shortkey:  val,
		}
		dk, _ := ParseKeyComment(ck.Comment)
		if dk == nil {
			c.adderr(group, key, "Shortkey ParseKeyComment: desc failed")
		} else {
			sk.Desc = dk.Text
		}
		e := tagInt(tag.Get("action"), func(id int) {
			sk.ActionID = id
		})
		c.testerr(e, group, key, "Shortkey action")

		elem.Set(reflect.ValueOf(sk))
		c.shortkeys = append(c.shortkeys, sk)

	case cdtype.Template:
		name := ck.String()
		if name == "" {
			name = tag.Get("default")
		}
		tmpl, e := cdtype.NewTemplate(name, c.appdir)
		if c.log.Err(e, "config load template", name) {
			elem.Set(reflect.ValueOf(cdtype.Template{FilePath: name}))
		} else {
			elem.Set(reflect.ValueOf(*tmpl))
		}

	case cdtype.ThemeExtra:
		sources := []string{"", "", ""} // system dir, local hint, distant hint.
		dk, _ := ParseKeyComment(ck.Comment)
		if dk != nil && len(dk.AuthorizedValues) > 2 {
			sources = dk.AuthorizedValues
		}
		theme, e := cdtype.NewThemeExtra(c.log,
			ck.String(), tag.Get("default"),
			c.confdir, c.appdir,
			sources[0], sources[1], sources[2],
		)
		if !c.testerr(e, group, key, "ThemeExtra path") {
			elem.Set(reflect.ValueOf(*theme))
			c.log.Debug("ThemeExtra", theme.Path())
		}

	default:
		return false
	}
	return true
}

// tagInt makes the call only if there is a valid value and returns parsing errors.
func tagInt(str string, call func(int)) error {
	if str == "" {
		return nil
	}
	val, e := strconv.Atoi(str)
	if e == nil {
		call(val)
	}
	return e
}

func (c *Config) testerr(e error, group, key, msg string, args ...interface{}) bool {
	if e == nil {
		return false
	}
	c.adderr(group, key, msg+": %s", append(args, e.Error())...)
	return true
}

func (c *Config) adderr(group, key, msg string, args ...interface{}) {
	args = append([]interface{}{group, key}, args...)
	c.Errors = append(c.Errors, fmt.Errorf("config: %s / %s -- "+msg, args...))
}

//-----------------------------------------------------------------[ MARSHAL ]--

// MarshalGroup fills the config with data from the struct provided.
//
// The group param must match a group in the file with the format [MYGROUP]
//
func (c *Config) MarshalGroup(v interface{}, group string, fieldKey cdtype.GetFieldKey) error {
	conf := reflect.ValueOf(v).Elem()
	typ := conf.Type()
	for i := 0; i < typ.NumField(); i++ { // Parsing all fields in type.
		elem := conf.Field(i)
		key := fieldKey(typ.Field(i))
		if key == "" || key == "-" || !elem.CanInterface() { // Disabled config key.
			continue
		}
		// tag := typ.Field(i).Tag
		e := keyset(c.Section(group).Key(key), elem.Interface())
		if e != nil {
			return e
		}
	}
	return nil
}

//--------------------------------------------------------------[ NEW CONFIG ]--

// Set sets a config value.
//
func (c *Config) Set(group, key string, uncast interface{}) error {
	return keyset(c.Section(group).Key(key), uncast)
}

func keyset(todisk *ini.Key, uncast interface{}) error {
	switch v := uncast.(type) {
	case string:
		todisk.SetValue(v)

	case []byte:
		todisk.SetValue(string(v))

	case bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:

		todisk.SetValue(fmt.Sprint(v))

	default:
		return fmt.Errorf("config.Set unknown type: %T", uncast)

	}
	return nil
}

// SetComment sets the comment for the key.
//
func (c *Config) SetComment(group, key, comment string) error {
	sect, e := c.File.GetSection(group)
	if e != nil {
		return e
	}
	ck, e := sect.GetKey(key)
	if e != nil {
		return e
	}
	ck.Comment = comment
	return nil
}

// GetComment gets the comment for the key.
//
func (c *Config) GetComment(group, key string) (string, error) {
	sect, e := c.File.GetSection(group)
	if e != nil {
		return "", e
	}
	ck, e := sect.GetKey(key)
	if e != nil {
		return "", e
	}
	return ck.Comment, nil
}

//
//-----------------------------------------------------------------[ VERSION ]--

// SetFileVersion replaces the version in a config file.
// The given group must represent the first group of the file.
//
func SetFileVersion(log cdtype.Logger, filename, group, oldver, newver string) error {
	return SetToFile(log, filename, func(cfg cdtype.ConfUpdater) error {
		return cfg.SetNewVersion(group, oldver, newver)
	})
}

// SetNewVersion replaces the version in a config file.
// The given group must represent the first group of the file.
//
func (c *Config) SetNewVersion(group, oldver, newver string) error {
	comment := c.Section(group).Comment
	prefix := "#" + oldver
	if !strings.HasPrefix(comment, prefix) {
		return errors.New("config.NewVersion: old version not found")
	}
	c.Section(group).Comment = strings.Replace(comment, prefix, "#"+newver, 1)
	return nil
}

//
//---------------------------------------------------------------[ CONFKEYER ]--

// confKey implements cdtype.ConfKeyer
type confKey struct {
	key *ini.Key
	valuer.Valuer
}

func (ck *confKey) Name() string    { return ck.key.Name() }
func (ck *confKey) Comment() string { return ck.key.Comment }

//
//------------------------------------------------------------------[ VALUER ]--

// value gives access to a storage group/key value. Implements cftype.Valuer
//
type value struct {
	c     Config
	group string
	name  string
	Err   error
}

// Get assigns the value to the given pointer to value (of the matching type).
//
func (o *value) Get(ptr interface{}) {
	switch v := ptr.(type) {
	case *bool:
		*v = o.Bool()

	case *int:
		*v = o.Int()

	case *float64:
		*v = o.Float()

	case *string:
		*v = o.String()

	case *[]bool:
		*v = o.ListBool()

	case *[]int:
		*v = o.ListInt()

	case *[]float64:
		*v = o.ListFloat()

	case *[]string:
		*v = o.ListString()
	}
}

// Bool returns the value as bool.
func (o *value) Bool() (v bool) {
	v, o.Err = o.c.Section(o.group).Key(o.name).Bool()
	return v
}

// Int returns the value as int.
func (o *value) Int() (v int) {
	v, o.Err = o.c.Section(o.group).Key(o.name).Int()
	return v
}

// Float returns the value as float64.
func (o *value) Float() (v float64) {
	v, o.Err = o.c.Section(o.group).Key(o.name).Float64()
	return v
}

// String returns the value as string.
func (o *value) String() (v string) {
	return o.c.Section(o.group).Key(o.name).String()
}

// ListBool returns the value as list of bool.
func (o *value) ListBool() (v []bool) {
	for _, tob := range o.ListString() {
		v = append(v, tob == "1" || tob == "true")
	}
	return v
	// return	o.c.Section(o.group).Key(o.name).
}

// ListInt returns the value as list of int.
func (o *value) ListInt() (v []int) {
	return o.c.Section(o.group).Key(o.name).Ints(";")
}

// ListFloat returns the value as list of float64.
func (o *value) ListFloat() (v []float64) {
	return o.c.Section(o.group).Key(o.name).Float64s(";")
}

// ListString returns the value as list of string.
func (o *value) ListString() (v []string) {
	return o.c.Section(o.group).Key(o.name).Strings(";")
}

// Set sets the pointed keyfile key value.
func (o *value) Set(v interface{}) { keyset(o.c.Section(o.group).Key(o.name), v) }

// Sprint returns the value as printable text.
func (o *value) Sprint() string {
	return o.String()
}

// SprintI returns the value as printable text of the element at position I in
// the list if possible.
//
func (o *value) SprintI(id int) string {
	list := o.ListString()
	if id >= len(list) {
		println("valuer SprintI. out of range:", id, list)
		return ""
	}
	return list[id]
}

// Count returns the number of elements in the list.
//
func (o *value) Count() int { return len(o.ListString()) } // unsure.

// MarshalGroup marshals a struct to a config group.
//
// func (c *Config) MarshalGroup(group string, v interface{}) error {
// 	return c.Section(group).ReflectFrom(v)
// }

//
//----------------------------------------------------------------[ CONF KEY ]--

// Modifier to show a widget according to the display backend.
const (
	ParseFlagCairoOnly  = '*'
	ParseFlagOpenGLOnly = '&'
)

// KeyBase defines a configuration entry options parsed from comment.
//
type KeyBase struct {
	NbElements        int                  // number of values stored in the key.
	AuthorizedValues  []string             //
	Text              string               // label for the key.
	Tooltip           string               // mouse over tooltip text.
	IsAlignedVertical bool                 // orientation for the key widget box.
	DisplayMode       cdglobal.DisplayMode // Dock display backend: all, cairo or opengl.
}

// ParseKeyComment parse comments for a key.
//
func ParseKeyComment(keyComment string) (*KeyBase, byte) {
	comment := strings.TrimLeft(keyComment, "# \n") // remove #, spaces, and endline from start.
	comment = strings.TrimRight(comment, "\n")      // remove endline from end.

	// Drop invalid or too short comments.
	if len(keyComment) < 2 || len(comment) == 0 || comment[0] == '[' {
		// '[' : on gere le bug de la Glib, qui rajoute les nouvelles cles apres le commentaire du groupe suivant !
		// log.DEV("LIBC BUG, DETECTED", comment) // often seem to be a [gtk-convert]
		return nil, ' '
	}

	typ := comment[0]
	comment = comment[1:]

	kb := &KeyBase{}
	comment = kb.parseBase(comment)

	if kb.NbElements == 0 {
		kb.NbElements = 1
	}

	comment = strings.TrimLeft(comment, "]1234567890") // Remove last bits of possible arguments.
	comment = strings.TrimLeft(comment, " ")           // Remove separator.

	// Special widget alignment with a trailing slash.
	if strings.HasSuffix(comment, "/") {
		comment = strings.TrimSuffix(comment, "/")
		kb.IsAlignedVertical = true
	}

	// Get tooltip.
	toolStart := strings.IndexByte(comment, '{')
	toolEnd := strings.IndexByte(comment, '}')
	if toolStart > 0 && toolEnd > 0 && toolStart < toolEnd {
		kb.Tooltip = comment[toolStart+1 : toolEnd]
		comment = comment[:toolStart-1]
	}

	kb.Text = strings.TrimRight(comment, "\n")

	return kb, typ
}

func (kb *KeyBase) parseBase(comment string) string {
	for i, c := range comment {
		switch c {
		case ' ', '-', '+':
			// Space and display flags: do nothing.

		case ParseFlagCairoOnly:
			kb.DisplayMode = cdglobal.DisplayModeCairo

		case ParseFlagOpenGLOnly:
			kb.DisplayMode = cdglobal.DisplayModeOpenGL

		default:
			// Try to detect a value indicating the number of elements.
			kb.NbElements, _ = strconv.Atoi(string(comment[i:]))

			// Try to get authorized values between square brackets.
			if c == '[' {
				values := comment[i+1 : strings.Index(comment, "]")]
				i += len(values) + 1

				kb.AuthorizedValues = strings.Split(values, ";")
			}

			// End of arguments at the start .
			return comment[i:]
		}
	}
	return ""
}
