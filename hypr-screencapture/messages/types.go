package messages

type Kind int

const (
	ScOn Kind = iota
	ScOff
)

type Msg struct{ Kind Kind }
