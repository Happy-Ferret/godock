package NetActivity

import "github.com/sqp/godock/libs/cdtype"

const defaultUpdateDelay = 3
const historyFile = "appdata/uptoshare_history.txt"

const (
	// EmblemAction is the position of the "upload in progress" emblem.
	EmblemAction = cdtype.EmblemTopRight

	// EmblemDownload is the position of the "download in progress" emblem.
	EmblemDownload = cdtype.EmblemTopLeft
)

// Commands references.
const (
	cmdLeft = iota
	cmdMiddle
)

//------------------------------------------------------------------[ CONFIG ]--

type appletConf struct {
	groupIcon          `group:"Icon"`
	groupConfiguration `group:"Configuration"`
	groupUpload        `group:"Upload"`
	groupActions       `group:"Actions"`
}

type groupIcon struct {
	Name string `conf:"name"`
}

type groupConfiguration struct {
	DisplayText   int
	DisplayValues int

	GaugeName string
	// GaugeRotate bool

	GraphType cdtype.RendererGraphType
	// GraphColorHigh []float64
	// GraphColorLow  []float64
	// GraphColorBg   []float64
	// GraphMix bool

	MonitoringEnabled bool
	UpdateDelay       int
	Devices           []string

	VideoDLEnabled   bool
	VideoDLPath      string
	VideoDLQuality   int
	VideoDLBlacklist []string
}

type groupUpload struct {
	DialogEnabled   bool
	DialogDuration  int32
	UploadHistory   int
	UploadRateLimit int

	FileForAll    bool
	SiteText      string
	SiteImage     string
	SiteVideo     string
	SiteFile      string
	PostAnonymous bool
}

type groupActions struct {
	LeftAction    int
	LeftCommand   string
	LeftClass     string
	MiddleAction  int
	MiddleCommand string

	// Still hidden.
	Debug bool
}

/*
Dropped from conf as there is no way to implement it from an external applet ATM.

#l+[No;With dock orientation;Yes] Rotate applet theme :
GaugeRotate=1

#c+ High value's colour :
#{It's the colour of the graphic for high rate values.}
GraphColorHigh=1;0;0;

#c+ Low value's colour :
#{Graph colour for low rate vaues:}
GraphColorLow=1;1;0;

#C+ Background colour of the graphic :
GraphColorBg=0.61438925764858476;0.61438925764858476;0.64814221408407724;0.7686274509803922;
*/
