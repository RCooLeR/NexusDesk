package shell

import (
	"fmt"
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

const dataGridPageStep = 25
const dataGridBoundaryStep = 1_000_000
const dataGridWidthSampleInitialRows = 32
const dataGridWidthSampleMaxRowsDefault = 160
const dataGridWidthSampleMaxRowsDense = 112
const dataGridWidthSampleMaxRowsUltraDense = 80
const dataGridWidthSampleMaxRowsWide = 96
const dataGridWidthMin = 120
const dataGridWidthMax = 420
const dataGridWidthMaxDense = 320
const dataGridWidthMaxUltraDense = 260
const dataGridDenseRowThreshold = 2000
const dataGridUltraDenseRowThreshold = 6000
const dataGridDenseColumnThreshold = 16
const dataGridWideColumnThreshold = 24
const dataGridUltraWideColumnThreshold = 48
const dataGridUltraWideColumnWidth = 180
const dataGridMaxVisibleColumns = 128

type dataGridRenderPolicy struct {
	Truncation  fyne.TextTruncation
	MinWidth    float32
	MaxWidth    float32
	HideDivider bool
}

func (v *View) setDataRowsText(text string) {
	if v.dataRowsDetail == nil {
		return
	}
	v.dataRowsColumns = nil
	v.dataRowsValues = nil
	v.dataRowsSelectedRow = -1
	v.dataRowsSelectedCol = -1
	v.dataRowsSampledRows = 0
	v.dataRowsOriginalRows = 0
	v.dataRowsClippedColumns = 0
	v.dataRowsDetail.SetText(text)
	v.setDataRowsStatus(dataRowsTextStatusLine(text))
	if v.dataRowsContainer == nil {
		return
	}
	v.dataRowsContainer.Objects = []fyne.CanvasObject{v.dataRowsDetail}
	v.dataRowsContainer.Refresh()
}

func (v *View) setDataRowsGrid(columns []string, rows [][]string) {
	if v.dataRowsContainer == nil {
		return
	}
	if len(columns) == 0 {
		v.setDataRowsText("")
		return
	}
	hadSelection := v.dataRowsSelectedRow >= 0 && v.dataRowsSelectedCol >= 0
	v.dataRowsColumns = append([]string{}, columns...)
	v.dataRowsValues = cloneDataRows(rows)
	visibleColumns, visibleRows, clippedColumns := limitDataGridColumns(v.dataRowsColumns, v.dataRowsValues, dataGridMaxVisibleColumns)
	v.dataRowsColumns = visibleColumns
	v.dataRowsValues = visibleRows
	v.dataRowsSelectedRow = -1
	v.dataRowsSelectedCol = -1
	v.dataRowsSampledRows = 0
	v.dataRowsOriginalRows = len(rows)
	v.dataRowsClippedColumns = clippedColumns
	v.dataRowsRenderPolicy = resolveDataGridRenderPolicy(v.dataRowsColumns, v.dataRowsValues)
	renderPolicy := v.dataRowsRenderPolicy
	widths, sampledRows := estimateDataGridColumnWidthsWithSampling(v.dataRowsColumns, v.dataRowsValues)
	v.dataRowsSampledRows = sampledRows
	v.ensureDataRowsTable()
	if v.dataRowsTable == nil {
		return
	}
	v.dataRowsTable.HideSeparators = renderPolicy.HideDivider
	if hadSelection {
		v.dataRowsTable.UnselectAll()
	}
	for _, index := range changedDataGridColumnWidthIndexes(v.dataRowsColumnWidths, widths) {
		v.dataRowsTable.SetColumnWidth(index, widths[index])
	}
	v.dataRowsColumnWidths = append([]float32{}, widths...)
	if !isSingleDataRowsTableObject(v.dataRowsContainer.Objects, v.dataRowsTable) {
		v.dataRowsContainer.Objects = []fyne.CanvasObject{v.dataRowsTable}
		v.dataRowsContainer.Refresh()
	}
	v.dataRowsTable.Refresh()
	v.applyDataGridSamplingStatusHint(sampledRows, len(rows), clippedColumns, renderPolicy)
	v.refreshDataRowsStatus()
}

func (v *View) ensureDataRowsTable() {
	if v.dataRowsTable != nil {
		return
	}
	table := widget.NewTable(
		func() (int, int) {
			return len(v.dataRowsValues) + 1, len(v.dataRowsColumns)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Truncation = v.dataRowsRenderPolicy.Truncation
			return label
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)
			label.Truncation = v.dataRowsRenderPolicy.Truncation
			if id.Row == 0 {
				label.TextStyle = fyne.TextStyle{Bold: true}
				if id.Col < len(v.dataRowsColumns) {
					label.SetText(v.dataRowsColumns[id.Col])
				} else {
					label.SetText("")
				}
				return
			}
			label.TextStyle = fyne.TextStyle{}
			rowIndex := id.Row - 1
			value := ""
			if rowIndex >= 0 && rowIndex < len(v.dataRowsValues) && id.Col < len(v.dataRowsValues[rowIndex]) {
				value = v.dataRowsValues[rowIndex][id.Col]
			}
			label.SetText(value)
		},
	)
	table.OnSelected = func(id widget.TableCellID) {
		v.dataRowsSelectedRow = id.Row
		v.dataRowsSelectedCol = id.Col
		v.refreshDataRowsStatus()
	}
	table.StickyRowCount = 1
	v.dataRowsTable = table
}

func (v *View) setDataRowsStatus(text string) {
	if v.dataRowsStatus == nil {
		return
	}
	v.dataRowsStatus.SetText(text)
}

func (v *View) refreshDataRowsStatus() {
	if v.dataRowsStatus == nil {
		return
	}
	v.dataRowsStatus.SetText(dataGridStatusLine(
		len(v.dataRowsValues),
		v.dataRowsOriginalRows,
		len(v.dataRowsColumns),
		v.dataRowsClippedColumns,
		v.dataRowsSelectedRow,
		v.dataRowsSelectedCol,
		v.dataRowsSampledRows,
		v.dataRowsRenderPolicy,
	))
}

func (v *View) copySelectedDataCell() {
	if !v.copySelectedDataCellToClipboard() {
		v.dataProfileStatus.SetText("Select a grid cell in Rows before copying.")
		return
	}
	v.dataProfileStatus.SetText("Copied selected grid cell.")
}

func (v *View) copySelectedDataRow() {
	value, ok := dataGridRowTSV(v.dataRowsColumns, v.dataRowsValues, v.dataRowsSelectedRow)
	if !ok {
		v.dataProfileStatus.SetText("Select a grid row in Rows before copying.")
		return
	}
	v.setClipboardContent(value)
	v.dataProfileStatus.SetText("Copied selected grid row as TSV.")
}

func (v *View) copySelection() {
	if !v.isBottomTabSelected("Data") {
		return
	}
	if !v.copySelectedDataCellToClipboard() {
		v.dataProfileStatus.SetText("Select a grid cell in Rows before copying.")
		return
	}
	v.dataProfileStatus.SetText("Copied selected grid cell.")
}

func (v *View) copySelectedDataCellToClipboard() bool {
	value, ok := dataGridCellValue(v.dataRowsColumns, v.dataRowsValues, v.dataRowsSelectedRow, v.dataRowsSelectedCol)
	if !ok {
		return false
	}
	v.setClipboardContent(value)
	return true
}

func (v *View) navigateDataGridSelection(rowDelta int, colDelta int) {
	if !v.isBottomTabSelected("Data") {
		return
	}
	if v.dataRowsTable == nil {
		return
	}
	nextRow, nextCol, ok := dataGridNextSelection(
		v.dataRowsSelectedRow,
		v.dataRowsSelectedCol,
		len(v.dataRowsValues)+1,
		len(v.dataRowsColumns),
		rowDelta,
		colDelta,
	)
	if !ok {
		return
	}
	cell := widget.TableCellID{Row: nextRow, Col: nextCol}
	v.dataRowsTable.Select(cell)
	v.dataRowsTable.ScrollTo(cell)
}

func (v *View) navigateDataGridPage(step int) {
	v.navigateDataGridSelection(step, 0)
}

func (v *View) navigateDataGridTop() {
	v.navigateDataGridSelection(-dataGridBoundaryStep, 0)
}

func (v *View) navigateDataGridBottom() {
	v.navigateDataGridSelection(dataGridBoundaryStep, 0)
}

func (v *View) navigateDataGridRowStart() {
	v.navigateDataGridSelection(0, -dataGridBoundaryStep)
}

func (v *View) navigateDataGridRowEnd() {
	v.navigateDataGridSelection(0, dataGridBoundaryStep)
}

func (v *View) setClipboardContent(value string) {
	if app := fyne.CurrentApp(); app != nil && app.Clipboard() != nil {
		app.Clipboard().SetContent(value)
		return
	}
	if v.window != nil {
		v.window.Clipboard().SetContent(value)
	}
}

func cloneDataRows(rows [][]string) [][]string {
	if len(rows) >= dataGridDenseRowThreshold {
		return append([][]string{}, rows...)
	}
	cloned := make([][]string, len(rows))
	for rowIndex := range rows {
		cloned[rowIndex] = append([]string{}, rows[rowIndex]...)
	}
	return cloned
}

func dataGridCellValue(columns []string, rows [][]string, selectedRow int, selectedCol int) (string, bool) {
	if len(columns) == 0 || selectedCol < 0 || selectedCol >= len(columns) || selectedRow < 0 {
		return "", false
	}
	if selectedRow == 0 {
		return columns[selectedCol], true
	}
	rowIndex := selectedRow - 1
	if rowIndex < 0 || rowIndex >= len(rows) {
		return "", false
	}
	if selectedCol >= len(rows[rowIndex]) {
		return "", true
	}
	return rows[rowIndex][selectedCol], true
}

func dataGridRowTSV(columns []string, rows [][]string, selectedRow int) (string, bool) {
	if len(columns) == 0 || selectedRow < 0 {
		return "", false
	}
	if selectedRow == 0 {
		return strings.Join(columns, "\t"), true
	}
	rowIndex := selectedRow - 1
	if rowIndex < 0 || rowIndex >= len(rows) {
		return "", false
	}
	values := make([]string, len(columns))
	for index := range columns {
		if index < len(rows[rowIndex]) {
			values[index] = rows[rowIndex][index]
		}
	}
	return strings.Join(values, "\t"), true
}

func dataGridNextSelection(selectedRow int, selectedCol int, rowCount int, colCount int, rowDelta int, colDelta int) (int, int, bool) {
	if rowCount <= 0 || colCount <= 0 {
		return 0, 0, false
	}
	if selectedRow < 0 || selectedRow >= rowCount || selectedCol < 0 || selectedCol >= colCount {
		return defaultDataGridSelection(rowCount), 0, true
	}
	nextRow := selectedRow + rowDelta
	nextCol := selectedCol + colDelta
	if nextRow < 0 {
		nextRow = 0
	}
	if nextRow >= rowCount {
		nextRow = rowCount - 1
	}
	if nextCol < 0 {
		nextCol = 0
	}
	if nextCol >= colCount {
		nextCol = colCount - 1
	}
	return nextRow, nextCol, true
}

func defaultDataGridSelection(rowCount int) int {
	if rowCount > 1 {
		return 1
	}
	return 0
}

func estimateDataGridColumnWidths(columns []string, rows [][]string) []float32 {
	widths, _ := estimateDataGridColumnWidthsWithSampling(columns, rows)
	return widths
}

func estimateDataGridColumnWidthsWithSampling(columns []string, rows [][]string) ([]float32, int) {
	renderPolicy := resolveDataGridRenderPolicy(columns, rows)
	if len(columns) >= dataGridUltraWideColumnThreshold {
		return fixedDataGridColumnWidths(columns, renderPolicy), 0
	}
	sampleBudget := dataGridWidthSampleBudget(len(rows), len(columns))
	sampledIndexes := sampledDataGridRowIndexes(len(rows), sampleBudget)
	widths := make([]float32, len(columns))
	for index, column := range columns {
		widths[index] = clampDataGridWidth(estimateCellWidth(column), renderPolicy.MinWidth, renderPolicy.MaxWidth)
	}
	for _, rowIndex := range sampledIndexes {
		if allDataGridWidthsAtMax(widths, renderPolicy.MaxWidth) {
			break
		}
		row := rows[rowIndex]
		for columnIndex := range columns {
			if columnIndex >= len(row) {
				continue
			}
			width := clampDataGridWidth(estimateCellWidth(row[columnIndex]), renderPolicy.MinWidth, renderPolicy.MaxWidth)
			if width > widths[columnIndex] {
				widths[columnIndex] = width
			}
		}
	}
	return widths, len(sampledIndexes)
}

func allDataGridWidthsAtMax(widths []float32, maxWidth float32) bool {
	for _, width := range widths {
		if width < maxWidth {
			return false
		}
	}
	return len(widths) > 0
}

func fixedDataGridColumnWidths(columns []string, policy dataGridRenderPolicy) []float32 {
	widths := make([]float32, len(columns))
	for index, column := range columns {
		headerWidth := estimateCellWidth(column)
		fixed := clampDataGridWidth(float32(dataGridUltraWideColumnWidth), policy.MinWidth, policy.MaxWidth)
		if headerWidth > fixed {
			fixed = clampDataGridWidth(headerWidth, policy.MinWidth, policy.MaxWidth)
		}
		widths[index] = fixed
	}
	return widths
}

func clampDataGridWidth(value float32, minWidth float32, maxWidth float32) float32 {
	if value < minWidth {
		return minWidth
	}
	if value > maxWidth {
		return maxWidth
	}
	return value
}

func changedDataGridColumnWidthIndexes(previous []float32, next []float32) []int {
	if len(next) == 0 {
		return nil
	}
	if len(previous) != len(next) {
		indexes := make([]int, len(next))
		for index := range next {
			indexes[index] = index
		}
		return indexes
	}
	indexes := make([]int, 0, len(next))
	for index := range next {
		if math.Abs(float64(previous[index]-next[index])) < 0.5 {
			continue
		}
		indexes = append(indexes, index)
	}
	return indexes
}

func isSingleDataRowsTableObject(objects []fyne.CanvasObject, table *widget.Table) bool {
	if table == nil || len(objects) != 1 {
		return false
	}
	return objects[0] == table
}

func resolveDataGridRenderPolicy(columns []string, rows [][]string) dataGridRenderPolicy {
	policy := dataGridRenderPolicy{
		Truncation:  fyne.TextTruncateEllipsis,
		MinWidth:    dataGridWidthMin,
		MaxWidth:    dataGridWidthMax,
		HideDivider: false,
	}
	rowCount := len(rows)
	colCount := len(columns)
	if rowCount >= dataGridDenseRowThreshold || colCount >= dataGridDenseColumnThreshold {
		policy.Truncation = fyne.TextTruncateClip
		policy.MaxWidth = dataGridWidthMaxDense
		policy.HideDivider = true
	}
	if rowCount >= dataGridUltraDenseRowThreshold {
		policy.MaxWidth = dataGridWidthMaxUltraDense
	}
	return policy
}

func dataGridWidthSampleBudget(rowCount int, colCount int) int {
	budget := dataGridWidthSampleMaxRowsDefault
	if rowCount >= dataGridUltraDenseRowThreshold {
		budget = dataGridWidthSampleMaxRowsUltraDense
	} else if rowCount >= dataGridDenseRowThreshold {
		budget = dataGridWidthSampleMaxRowsDense
	}
	if colCount >= dataGridWideColumnThreshold && budget > dataGridWidthSampleMaxRowsWide {
		budget = dataGridWidthSampleMaxRowsWide
	}
	if budget < dataGridWidthSampleInitialRows {
		budget = dataGridWidthSampleInitialRows
	}
	return budget
}

func (v *View) applyDataGridSamplingStatusHint(sampledRows int, totalRows int, clippedColumns int, policy dataGridRenderPolicy) {
	if v.dataProfileStatus == nil {
		return
	}
	base := strings.TrimSpace(v.dataProfileStatus.Text)
	if base == "" {
		return
	}
	v.dataProfileStatus.SetText(withDataGridSamplingStatusHint(base, sampledRows, totalRows, clippedColumns, policy.HideDivider))
}

func withDataGridSamplingStatusHint(base string, sampledRows int, totalRows int, clippedColumns int, denseMode bool) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	if marker := strings.Index(base, " | Grid: sampled "); marker >= 0 {
		base = strings.TrimSpace(base[:marker])
	}
	hint := fmt.Sprintf("Grid: sampled %d/%d rows", sampledRows, totalRows)
	if denseMode {
		hint += ", dense mode"
	}
	if totalRows > 0 && sampledRows == 0 {
		hint = "Grid: header-width mode"
	}
	if clippedColumns > 0 {
		hint += fmt.Sprintf(", showing first %d columns", dataGridMaxVisibleColumns)
	}
	return base + " | " + hint
}

func dataRowsTextStatusLine(text string) string {
	if strings.TrimSpace(text) == "" {
		return "Rows: no grid loaded."
	}
	return "Rows: text preview loaded | Grid navigation unavailable for this result."
}

func dataGridStatusLine(
	visibleRows int,
	originalRows int,
	visibleColumns int,
	clippedColumns int,
	selectedRow int,
	selectedCol int,
	sampledRows int,
	policy dataGridRenderPolicy,
) string {
	if visibleColumns <= 0 {
		return "Rows: no grid loaded."
	}
	rowSummary := fmt.Sprintf("Rows: %d shown", visibleRows)
	if originalRows > 0 && originalRows != visibleRows {
		rowSummary = fmt.Sprintf("Rows: %d shown of %d", visibleRows, originalRows)
	}
	columnSummary := fmt.Sprintf("Columns: %d visible", visibleColumns)
	if clippedColumns > 0 {
		columnSummary += fmt.Sprintf(" (+%d hidden)", clippedColumns)
	}
	sizing := "Sizing: headers only"
	if sampledRows > 0 {
		totalRows := originalRows
		if totalRows <= 0 {
			totalRows = visibleRows
		}
		sizing = fmt.Sprintf("Sizing: sampled %d/%d rows", sampledRows, totalRows)
	}
	return strings.Join([]string{
		rowSummary,
		columnSummary,
		"Density: " + dataGridDensityLabel(policy),
		sizing,
		dataGridSelectionLabel(selectedRow, selectedCol),
		"Copy: Ctrl/Cmd+C cell, Copy row TSV",
	}, " | ")
}

func dataGridDensityLabel(policy dataGridRenderPolicy) string {
	if policy.HideDivider && policy.MaxWidth <= dataGridWidthMaxUltraDense {
		return "ultra dense"
	}
	if policy.HideDivider {
		return "dense"
	}
	return "standard"
}

func dataGridSelectionLabel(row int, col int) string {
	if row < 0 || col < 0 {
		return "Selection: none"
	}
	if row == 0 {
		return fmt.Sprintf("Selection: header C%d", col+1)
	}
	return fmt.Sprintf("Selection: R%d C%d", row, col+1)
}

func limitDataGridColumns(columns []string, rows [][]string, maxColumns int) ([]string, [][]string, int) {
	if maxColumns <= 0 || len(columns) <= maxColumns {
		return append([]string{}, columns...), rows, 0
	}
	clipped := len(columns) - maxColumns
	visibleColumns := append([]string{}, columns[:maxColumns]...)
	visibleRows := make([][]string, len(rows))
	for rowIndex := range rows {
		row := rows[rowIndex]
		if len(row) <= maxColumns {
			visibleRows[rowIndex] = row
			continue
		}
		visibleRows[rowIndex] = row[:maxColumns]
	}
	return visibleColumns, visibleRows, clipped
}

func sampledDataGridRowIndexes(totalRows int, sampleBudget int) []int {
	if totalRows <= 0 {
		return nil
	}
	if sampleBudget <= 0 {
		sampleBudget = dataGridWidthSampleMaxRowsDefault
	}
	if totalRows <= sampleBudget {
		indexes := make([]int, totalRows)
		for index := range indexes {
			indexes[index] = index
		}
		return indexes
	}
	indexes := make([]int, 0, sampleBudget)
	seen := make(map[int]struct{}, sampleBudget)
	addIndex := func(index int) {
		if index < 0 || index >= totalRows {
			return
		}
		if _, exists := seen[index]; exists {
			return
		}
		seen[index] = struct{}{}
		indexes = append(indexes, index)
	}
	initialRows := dataGridWidthSampleInitialRows
	if initialRows > totalRows {
		initialRows = totalRows
	}
	for index := 0; index < initialRows; index++ {
		addIndex(index)
	}
	remainingSlots := sampleBudget - len(indexes)
	if remainingSlots <= 0 {
		return indexes
	}
	start := initialRows
	if start >= totalRows {
		return indexes
	}
	remainingRows := totalRows - start
	if remainingRows <= remainingSlots {
		for index := start; index < totalRows; index++ {
			addIndex(index)
		}
		return indexes
	}
	if remainingSlots == 1 {
		addIndex(totalRows - 1)
		return indexes
	}
	span := float64(totalRows - 1 - start)
	steps := float64(remainingSlots - 1)
	for stepIndex := 0; stepIndex < remainingSlots; stepIndex++ {
		offset := int(math.Round(float64(stepIndex) * span / steps))
		addIndex(start + offset)
	}
	addIndex(totalRows - 1)
	return indexes
}

func estimateCellWidth(value string) float32 {
	compact := strings.Join(strings.Fields(value), " ")
	if compact == "" {
		return 120
	}
	length := len(compact)
	if length > 48 {
		length = 48
	}
	return float32(length*8 + 24)
}
