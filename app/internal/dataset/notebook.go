package dataset

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const notebookStoreName = "notebooks.json"
const maxSavedNotebooksPerDataset = 12

type NotebookCell struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
	SQL   string `json:"sql"`
}

type Notebook struct {
	ID        string         `json:"id"`
	RelPath   string         `json:"relPath"`
	Label     string         `json:"label"`
	Cells     []NotebookCell `json:"cells"`
	UpdatedAt string         `json:"updatedAt"`
}

type NotebookSaveRequest struct {
	ID      string         `json:"id"`
	RelPath string         `json:"relPath"`
	Label   string         `json:"label"`
	Cells   []NotebookCell `json:"cells"`
}

func SaveNotebook(root string, request NotebookSaveRequest) (Notebook, error) {
	absRoot, _, cleanRel, err := resolveDatasetPath(root, request.RelPath)
	if err != nil {
		return Notebook{}, err
	}

	now := time.Now().UTC()
	label := strings.TrimSpace(request.Label)
	if label == "" {
		label = "SQL Notebook"
	}
	id := cleanNotebookID(request.ID)
	if id == "" {
		id = newNotebookID(label, now)
	}
	cells := cleanNotebookCells(request.Cells)
	if len(cells) == 0 {
		cells = []NotebookCell{{ID: "cell-1", Kind: "sql", Label: "Cell 1", SQL: "select * from dataset limit 20"}}
	}

	items, err := readNotebooks(absRoot)
	if err != nil {
		return Notebook{}, err
	}

	saved := Notebook{
		ID:        id,
		RelPath:   filepath.ToSlash(cleanRel),
		Label:     label,
		Cells:     cells,
		UpdatedAt: now.Format(time.RFC3339),
	}
	items[notebookKey(saved.RelPath, saved.ID)] = saved
	items = trimNotebooks(items, saved.RelPath)

	if err := writeNotebooks(absRoot, items); err != nil {
		return Notebook{}, err
	}
	return saved, nil
}

func ListNotebooks(root string, relPath string) ([]Notebook, error) {
	absRoot, _, cleanRel, err := resolveDatasetPath(root, relPath)
	if err != nil {
		return nil, err
	}
	items, err := readNotebooks(absRoot)
	if err != nil {
		return nil, err
	}

	cleanRelPath := filepath.ToSlash(cleanRel)
	notebooks := []Notebook{}
	for _, item := range items {
		if item.RelPath == cleanRelPath {
			notebooks = append(notebooks, item)
		}
	}
	sort.SliceStable(notebooks, func(i, j int) bool {
		return notebooks[i].UpdatedAt > notebooks[j].UpdatedAt
	})
	return notebooks, nil
}

func readNotebooks(absRoot string) (map[string]Notebook, error) {
	path := filepath.Join(absRoot, filepath.FromSlash(profileDirRelPath), notebookStoreName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]Notebook{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := map[string]Notebook{}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func writeNotebooks(absRoot string, items map[string]Notebook) error {
	dir := filepath.Join(absRoot, filepath.FromSlash(profileDirRelPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, notebookStoreName), append(data, '\n'), 0o644)
}

func trimNotebooks(items map[string]Notebook, relPath string) map[string]Notebook {
	notebooks := []Notebook{}
	for _, item := range items {
		if item.RelPath == relPath {
			notebooks = append(notebooks, item)
		}
	}
	sort.SliceStable(notebooks, func(i, j int) bool {
		return notebooks[i].UpdatedAt > notebooks[j].UpdatedAt
	})
	for index, notebook := range notebooks {
		if index < maxSavedNotebooksPerDataset {
			continue
		}
		delete(items, notebookKey(notebook.RelPath, notebook.ID))
	}
	return items
}

func cleanNotebookCells(cells []NotebookCell) []NotebookCell {
	cleaned := []NotebookCell{}
	for index, cell := range cells {
		kind := strings.ToLower(strings.TrimSpace(cell.Kind))
		if kind != "chart" {
			kind = "sql"
		}
		id := strings.TrimSpace(cell.ID)
		if id == "" {
			id = "cell-" + strconvInt(index+1)
		}
		label := strings.TrimSpace(cell.Label)
		if label == "" {
			if kind == "chart" {
				label = "Chart " + strconvInt(index+1)
			} else {
				label = "Cell " + strconvInt(index+1)
			}
		}
		cleaned = append(cleaned, NotebookCell{
			ID:    id,
			Kind:  kind,
			Label: label,
			SQL:   strings.TrimSpace(cell.SQL),
		})
	}
	return cleaned
}

func newNotebookID(label string, now time.Time) string {
	base := strings.ToLower(strings.TrimSpace(label))
	base = notebookIDCleaner.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "notebook"
	}
	return base + "-" + now.Format("20060102150405")
}

func cleanNotebookID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = notebookIDCleaner.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func notebookKey(relPath string, id string) string {
	return strings.ToLower(relPath) + "\x00" + cleanNotebookID(id)
}

func strconvInt(value int) string {
	return strconv.Itoa(value)
}

var notebookIDCleaner = regexp.MustCompile(`[^a-z0-9_-]+`)
