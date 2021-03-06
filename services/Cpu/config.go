package Cpu

import "github.com/sqp/godock/libs/cdtype"

const (
	defaultUpdateDelay = 3
)

// Commands references.
const (
	cmdLeft = iota
	cmdMiddle
)

//------------------------------------------------------------------[ CONFIG ]--

type appletConf struct {
	cdtype.ConfGroupIconBoth `group:"Icon"`
	groupConfiguration       `group:"Configuration"`
	groupActions             `group:"Actions"`
}

type groupConfiguration struct {
	DisplayText   cdtype.InfoPosition
	DisplayValues int

	GaugeName string
	// GaugeRotate bool

	GraphType cdtype.RendererGraphType
	// GraphColorHigh []float64
	// GraphColorLow  []float64
	// GraphColorBg   []float64
	// GraphMix bool

	UpdateDelay cdtype.Duration `default:"3"`
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
