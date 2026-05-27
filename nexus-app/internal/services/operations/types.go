package operations

type FileKind string

const (
	FileKindDockerfile FileKind = "dockerfile"
	FileKindCompose    FileKind = "compose"
	FileKindEnv        FileKind = "env"
	FileKindConfig     FileKind = "config"
	FileKindLog        FileKind = "log"
	FileKindScript     FileKind = "script"
)

type File struct {
	RelPath string
	Name    string
	Kind    FileKind
	Size    int64
}

type Summary struct {
	Files       int
	Dockerfiles int
	Compose     int
	Env         int
	Config      int
	Logs        int
	Scripts     int
	SkippedDirs int
	SkippedSize int
	Unreadable  int
	EntryCap    int
}

type ScanResult struct {
	Files   []File
	Summary Summary
	Message string
}

type ComposeService struct {
	Name      string
	Image     string
	Ports     []string
	Volumes   []string
	DependsOn []string
}

type ComposeTopology struct {
	Summary      string
	Services     []ComposeTopologyService
	Edges        []ComposeTopologyEdge
	ExposedPorts []ComposePortExposure
	NamedVolumes []string
	Warnings     []string
}

type ComposeTopologyService struct {
	Name    string
	Image   string
	Ports   []string
	Volumes []string
}

type ComposeTopologyEdge struct {
	From     string
	To       string
	Relation string
	Missing  bool
}

type ComposePortExposure struct {
	Service string
	Port    string
}

type Inspection struct {
	File      File
	Text      string
	Truncated bool
	Services  []ComposeService
	Topology  ComposeTopology
	Warnings  []string
}
