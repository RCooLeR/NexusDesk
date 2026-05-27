package datasets

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

const (
	notebookStoreRelDir        = ".nexusdesk/datasets"
	notebookStoreName          = "notebooks.json"
	maxSavedNotebooksPerSource = 12
	maxNotebookCells           = 32
	maxNotebookLabelLength     = 120
	maxNotebookSQLLength       = 20000
)

var notebookIDPattern = regexp.MustCompile(`[^a-z0-9_-]+`)

func (s *Service) SaveNotebook(root string, request NotebookSaveRequest) (Notebook, error) {
	return SaveNotebook(root, request)
}

func (s *Service) ListNotebooks(root string, relPath string) ([]Notebook, error) {
	return ListNotebooks(root, relPath)
}

func SaveNotebook(root string, request NotebookSaveRequest) (Notebook, error) {
	_, cleanRel, _, err := resolveDatasetFile(root, request.RelPath)
	if err != nil {
		return Notebook{}, err
	}
	absRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return Notebook{}, err
	}
	now := time.Now().UTC()
	label := cleanNotebookLabel(request.Label)
	id := cleanNotebookID(request.ID)
	if id == "" {
		id = newNotebookID(label, now)
	}
	cells := cleanNotebookCells(request.Cells, now)
	if len(cells) == 0 {
		cells = []NotebookCell{{
			ID:        "cell-1",
			Kind:      "sql",
			Label:     "Cell 1",
			SQL:       "select * from dataset limit 20",
			CreatedAt: now,
			UpdatedAt: now,
		}}
	}

	items, err := readNotebooks(absRoot)
	if err != nil {
		return Notebook{}, err
	}
	key := notebookKey(cleanRel, id)
	created := now
	if existing, ok := items[key]; ok && !existing.CreatedAt.IsZero() {
		created = existing.CreatedAt
	}
	saved := Notebook{
		ID:        id,
		RelPath:   cleanRel,
		Label:     label,
		Cells:     cells,
		CreatedAt: created,
		UpdatedAt: now,
	}
	items[key] = saved
	trimNotebooks(items, cleanRel)
	if err := writeNotebooks(absRoot, items); err != nil {
		return Notebook{}, err
	}
	return saved, nil
}

func ListNotebooks(root string, relPath string) ([]Notebook, error) {
	_, cleanRel, _, err := resolveDatasetFile(root, relPath)
	if err != nil {
		return nil, err
	}
	absRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return nil, err
	}
	items, err := readNotebooks(absRoot)
	if err != nil {
		return nil, err
	}
	notebooks := []Notebook{}
	for _, item := range items {
		if filepath.ToSlash(item.RelPath) == cleanRel {
			notebooks = append(notebooks, item)
		}
	}
	sort.SliceStable(notebooks, func(i, j int) bool {
		return notebooks[i].UpdatedAt.After(notebooks[j].UpdatedAt)
	})
	return notebooks, nil
}

func readNotebooks(absRoot string) (map[string]Notebook, error) {
	path := filepath.Join(absRoot, filepath.FromSlash(notebookStoreRelDir), notebookStoreName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]Notebook{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := map[string]Notebook{}
	if len(strings.TrimSpace(string(data))) == 0 {
		return items, nil
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func writeNotebooks(absRoot string, items map[string]Notebook) error {
	dir := filepath.Join(absRoot, filepath.FromSlash(notebookStoreRelDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, notebookStoreName), append(data, '\n'), 0o644)
}

func cleanNotebookCells(cells []NotebookCell, now time.Time) []NotebookCell {
	cleaned := []NotebookCell{}
	for index, cell := range cells {
		if len(cleaned) >= maxNotebookCells {
			break
		}
		kind := strings.ToLower(strings.TrimSpace(cell.Kind))
		if kind != "chart" {
			kind = "sql"
		}
		sqlText := strings.TrimSpace(cell.SQL)
		if len(sqlText) > maxNotebookSQLLength {
			sqlText = sqlText[:maxNotebookSQLLength]
		}
		if kind == "sql" && sqlText == "" {
			continue
		}
		id := cleanNotebookID(cell.ID)
		if id == "" {
			id = "cell-" + strconv.Itoa(index+1)
		}
		created := cell.CreatedAt
		if created.IsZero() {
			created = now
		}
		updated := cell.UpdatedAt
		if updated.IsZero() {
			updated = now
		}
		cleaned = append(cleaned, NotebookCell{
			ID:        id,
			Kind:      kind,
			Label:     cleanNotebookCellLabel(cell.Label, kind, index+1),
			SQL:       sqlText,
			CreatedAt: created,
			UpdatedAt: updated,
		})
	}
	return cleaned
}

func trimNotebooks(items map[string]Notebook, relPath string) {
	sourceItems := []Notebook{}
	for _, item := range items {
		if filepath.ToSlash(item.RelPath) == relPath {
			sourceItems = append(sourceItems, item)
		}
	}
	sort.SliceStable(sourceItems, func(i, j int) bool {
		return sourceItems[i].UpdatedAt.After(sourceItems[j].UpdatedAt)
	})
	for index, item := range sourceItems {
		if index < maxSavedNotebooksPerSource {
			continue
		}
		delete(items, notebookKey(item.RelPath, item.ID))
	}
}

func cleanNotebookLabel(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "SQL Notebook"
	}
	if len(value) > maxNotebookLabelLength {
		return value[:maxNotebookLabelLength]
	}
	return value
}

func cleanNotebookCellLabel(value string, kind string, index int) string {
	value = cleanNotebookLabel(value)
	if value != "SQL Notebook" {
		return value
	}
	if kind == "chart" {
		return "Chart " + strconv.Itoa(index)
	}
	return "Cell " + strconv.Itoa(index)
}

func newNotebookID(label string, now time.Time) string {
	base := cleanNotebookID(label)
	if base == "" {
		base = "notebook"
	}
	return base + "-" + now.Format("20060102150405")
}

func cleanNotebookID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = notebookIDPattern.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func notebookKey(relPath string, id string) string {
	return strings.ToLower(filepath.ToSlash(relPath)) + "\x00" + cleanNotebookID(id)
}
