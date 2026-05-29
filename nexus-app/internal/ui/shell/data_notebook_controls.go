package shell

import (
	"fmt"
	"strings"

	datasetsSvc "nexusdesk/internal/services/datasets"
)

func (v *View) refreshNotebookCellSelector() {
	if v.data.dataNotebookCellSelect == nil {
		return
	}
	cells := notebookCellsFromEditor(v.data.dataQueryEntry.Text)
	options := notebookCellOptions(cells)
	v.data.dataNotebookCellSelect.Options = options
	if len(options) == 0 {
		v.data.dataNotebookCellIndex = 0
		v.data.dataNotebookCellSelect.ClearSelected()
		v.data.dataNotebookCellSelect.Refresh()
		v.setDataSummary("# SQL Notebook Cells\n\nNo valid cells found in the editor.\n")
		return
	}
	if v.data.dataNotebookCellIndex < 0 {
		v.data.dataNotebookCellIndex = 0
	}
	if v.data.dataNotebookCellIndex >= len(options) {
		v.data.dataNotebookCellIndex = len(options) - 1
	}
	v.data.dataNotebookCellSelect.SetSelected(options[v.data.dataNotebookCellIndex])
	v.data.dataNotebookCellSelect.Refresh()
	v.setDataSummary(formatNotebookCellOutline(cells, v.data.dataNotebookCellIndex))
}

func (v *View) moveSelectedNotebookCell(delta int) {
	cells := notebookCellsFromEditor(v.data.dataQueryEntry.Text)
	nextCells, nextIndex, moved := moveNotebookCells(cells, v.data.dataNotebookCellIndex, delta)
	if !moved {
		v.data.dataProfileStatus.SetText("Notebook cell cannot move in that direction.")
		return
	}
	v.data.dataNotebookCellIndex = nextIndex
	v.data.dataQueryEntry.SetText(formatNotebookForEditor(datasetsSvc.Notebook{Cells: nextCells}))
	v.refreshNotebookCellSelector()
	v.data.dataProfileStatus.SetText(fmt.Sprintf("Moved notebook cell to position %d.", nextIndex+1))
}

func (v *View) deleteSelectedNotebookCell() {
	cells := notebookCellsFromEditor(v.data.dataQueryEntry.Text)
	nextCells, nextIndex, deleted := deleteNotebookCell(cells, v.data.dataNotebookCellIndex)
	if !deleted {
		v.data.dataProfileStatus.SetText("Keep at least one notebook cell before deleting.")
		return
	}
	v.data.dataNotebookCellIndex = nextIndex
	v.data.dataQueryEntry.SetText(formatNotebookForEditor(datasetsSvc.Notebook{Cells: nextCells}))
	v.refreshNotebookCellSelector()
	v.data.dataProfileStatus.SetText(fmt.Sprintf("Deleted notebook cell. %d cell(s) remain.", len(nextCells)))
}

func notebookCellOptions(cells []datasetsSvc.NotebookCell) []string {
	options := make([]string, 0, len(cells))
	for index, cell := range cells {
		options = append(options, fmt.Sprintf("%02d  %s [%s]", index+1, firstNonEmptyString(cell.Label, cell.ID), notebookCellKindLabel(cell.Kind)))
	}
	return options
}

func notebookCellOptionIndex(options []string, choice string) int {
	for index, option := range options {
		if option == choice {
			return index
		}
	}
	return 0
}

func moveNotebookCells(cells []datasetsSvc.NotebookCell, index int, delta int) ([]datasetsSvc.NotebookCell, int, bool) {
	if len(cells) == 0 || delta == 0 {
		return cells, index, false
	}
	if index < 0 || index >= len(cells) {
		index = 0
	}
	nextIndex := index + delta
	if nextIndex < 0 || nextIndex >= len(cells) {
		return cells, index, false
	}
	next := append([]datasetsSvc.NotebookCell(nil), cells...)
	next[index], next[nextIndex] = next[nextIndex], next[index]
	return renumberNotebookCells(next), nextIndex, true
}

func deleteNotebookCell(cells []datasetsSvc.NotebookCell, index int) ([]datasetsSvc.NotebookCell, int, bool) {
	if len(cells) <= 1 {
		return cells, index, false
	}
	if index < 0 || index >= len(cells) {
		index = len(cells) - 1
	}
	next := append([]datasetsSvc.NotebookCell{}, cells[:index]...)
	next = append(next, cells[index+1:]...)
	nextIndex := index
	if nextIndex >= len(next) {
		nextIndex = len(next) - 1
	}
	return renumberNotebookCells(next), nextIndex, true
}

func renumberNotebookCells(cells []datasetsSvc.NotebookCell) []datasetsSvc.NotebookCell {
	for index := range cells {
		cells[index].ID = fmt.Sprintf("cell-%d", index+1)
	}
	return cells
}

func formatNotebookCellOutline(cells []datasetsSvc.NotebookCell, activeIndex int) string {
	var builder strings.Builder
	builder.WriteString("# SQL Notebook Cells\n\n")
	for index, cell := range cells {
		marker := " "
		if index == activeIndex {
			marker = "*"
		}
		builder.WriteString(fmt.Sprintf("%s %d. %s [%s]\n", marker, index+1, firstNonEmptyString(cell.Label, cell.ID), notebookCellKindLabel(cell.Kind)))
		if strings.TrimSpace(cell.SQL) != "" {
			builder.WriteString("   ")
			builder.WriteString(compactDataLine(cell.SQL, 160))
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func notebookCellKindLabel(kind string) string {
	if strings.EqualFold(strings.TrimSpace(kind), "chart") {
		return "chart"
	}
	return "sql"
}
