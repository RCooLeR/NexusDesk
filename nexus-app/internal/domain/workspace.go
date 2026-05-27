package domain

type Workspace struct {
	Root    string
	Name    string
	Summary ScanSummary
	Tree    []WorkspaceNode
}

type ScanSummary struct {
	Included     int
	Ignored      int
	DepthSkipped int
	EntryCap     int
	Unreadable   int
}

type WorkspaceNode struct {
	ID       string
	ParentID string
	Name     string
	RelPath  string
	Kind     WorkspaceNodeKind
	Size     int64
	Ignored  bool
	Children []WorkspaceNode
}

type WorkspaceNodeKind string

const (
	NodeDirectory WorkspaceNodeKind = "directory"
	NodeFile      WorkspaceNodeKind = "file"
)

type FilePreview struct {
	RelPath   string
	Name      string
	Size      int64
	Kind      PreviewKind
	MediaType string
	Encoding  string
	Text      string
	Bytes     []byte
	Table     *TablePreview
	Document  *DocumentPreview
	PDF       *PDFPreview
}

type TablePreview struct {
	Headers   []string
	Rows      [][]string
	Delimiter string
	Sheet     string
	Sheets    []string
	Truncated bool
}

type DocumentPreview struct {
	Text      string
	Truncated bool
}

type PDFPreview struct {
	Pages     []TextPage
	Text      string
	Truncated bool
}

type TextPage struct {
	Page int
	Text string
}

type PreviewKind string

const (
	PreviewText   PreviewKind = "text"
	PreviewImage  PreviewKind = "image"
	PreviewTable  PreviewKind = "table"
	PreviewDoc    PreviewKind = "document"
	PreviewPDF    PreviewKind = "pdf"
	PreviewBinary PreviewKind = "binary"
)
