package vt10x

/*
type EventType int

const (
	Blank EventType = 1 << iota
	CharSet
	Clear
	ScrollUp
	ScrollDown
)

type Diff interface {
	EventType() EventType
}

type BlankEvent struct{}

func (e BlankEvent) EventType() EventType {
	return Blank
}

type CharSetEvent struct {
	Cells []Cell
}

type Cell struct {
	Glyph Glyph
	X     int
	Y     int
}

func (e CharSetEvent) EventType() EventType {
	return CharSet
}

func (e CharSetEvent) AddChar(glyph Glyph, x, y int) {
	cell := Cell{
		Glyph: glyph,
		X:     x,
		Y:     y,
	}
	e.Cells = append(e.Cells, cell)
}


func (e ClearEvent) EventType() EventType {
	return Clear
}
*/

type Cell struct {
	X int
	Y int
}

type CharZone struct {
	X0 int
	Y0 int
	X1 int
	Y1 int
}

func (z *CharZone) Width() int {
	return z.X1 - z.X0 + 1
}

func (z *CharZone) Height() int {
	return z.Y1 - z.Y0 + 1
}

type ClearBox struct {
	X0 int
	Y0 int
	X1 int
	Y1 int
}

type Diff struct {
	BackSpace bool
	Clear     bool
}

func NewDiff() Diff {
	return Diff{
		BackSpace: false,
		Clear:     false,
	}
}
