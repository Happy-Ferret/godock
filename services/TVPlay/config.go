package TVPlay

import (
	"github.com/sqp/godock/libs/cdtype"
	"github.com/sqp/gupnp/upnptype"
)

type appletConf struct {
	cdtype.ConfGroupIconBoth `group:"Icon"`
	groupConfiguration       `group:"Configuration"`
	groupActions             `group:"Actions"`
}

type groupConfiguration struct {
	VolumeDelta int
	SeekDelta   int

	PreferredRenderer string
	PreferredServer   string

	DialogEnabled bool
	DialogTimer   int
	AnimName      string
	AnimDuration  int

	WindowVisibility int
}

type groupActions struct {
	ActionClickMiddle string
	ActionMouseWheel  string

	ShortkeyMute         *cdtype.Shortkey `action:"1"`
	ShortkeyVolumeUp     *cdtype.Shortkey `action:"2"`
	ShortkeyVolumeDown   *cdtype.Shortkey `action:"3"`
	ShortkeyPlayPause    *cdtype.Shortkey `action:"4"`
	ShortkeyStop         *cdtype.Shortkey `action:"5"`
	ShortkeySeekBackward *cdtype.Shortkey `action:"6"`
	ShortkeySeekForward  *cdtype.Shortkey `action:"7"`
}

// Actions available in right click menu.
//
var dockMenu = []int{
	int(upnptype.ActionToggleMute),
	int(upnptype.ActionVolumeUp),
	int(upnptype.ActionVolumeDown),
	int(upnptype.ActionNone),
	int(upnptype.ActionPlayPause),
	int(upnptype.ActionStop),
	int(upnptype.ActionSeekBackward),
	int(upnptype.ActionSeekForward),
}
