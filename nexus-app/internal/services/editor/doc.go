package editor

type Kind string

const (
	KindWelcome     Kind = "welcome"
	KindFile        Kind = "file"
	KindPlaceholder Kind = "placeholder"
)

type Tab struct {
	ID      string
	Title   string
	RelPath string
	Kind    Kind
	Dirty   bool
	Pinned  bool
}
