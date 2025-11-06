package messages

type Kind int

const (
	ScOn Kind = iota
	ScOff
)

type Source int

const (
	SourceHyprland Source = iota
	SourcePipeWire
)

type Msg struct {
	Kind   Kind
	Source Source
}
