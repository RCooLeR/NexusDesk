package history

import "time"

type Kind string

const (
	KindChat     Kind = "chat"
	KindArtifact Kind = "artifact"
	KindJob      Kind = "job"
	KindAgent    Kind = "agent"
)

type Item struct {
	Kind        Kind
	Ref         string
	Title       string
	Summary     string
	Detail      string
	When        time.Time
	SourcePaths []string
}

type Options struct {
	Query string
	Kind  Kind
	Limit int
}
