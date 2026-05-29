package shell

import (
	"fmt"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func TestEstimateDataGridColumnWidthsRespectsBounds(t *testing.T) {
	widths := estimateDataGridColumnWidths(
		[]string{"short", "very_long_column_name"},
		[][]string{
			{"a", "this is a significantly longer cell value than the header"},
		},
	)
	if len(widths) != 2 {
		t.Fatalf("expected two widths, got %#v", widths)
	}
	if widths[0] < 120 || widths[0] > 420 {
		t.Fatalf("width 0 should be bounded, got %f", widths[0])
	}
	if widths[1] <= widths[0] {
		t.Fatalf("expected second column width to be larger, got %#v", widths)
	}
}

func TestEstimateCellWidthCapsVeryLongValues(t *testing.T) {
	short := estimateCellWidth("id")
	long := estimateCellWidth("this value is intentionally very very very very long to trigger cap")
	if long <= short {
		t.Fatalf("expected long width to exceed short width: short=%f long=%f", short, long)
	}
	if long > 420 {
		t.Fatalf("expected capped long width, got %f", long)
	}
}

func TestDataGridCellValueSupportsHeaderAndDataRows(t *testing.T) {
	columns := []string{"id", "name"}
	rows := [][]string{{"1", "Ada"}}
	header, ok := dataGridCellValue(columns, rows, 0, 1)
	if !ok || header != "name" {
		t.Fatalf("expected header value, got %q ok=%v", header, ok)
	}
	value, ok := dataGridCellValue(columns, rows, 1, 0)
	if !ok || value != "1" {
		t.Fatalf("expected row value, got %q ok=%v", value, ok)
	}
}

func TestDataGridRowTSVReturnsHeaderAndDataRows(t *testing.T) {
	columns := []string{"id", "name"}
	rows := [][]string{{"1", "Ada"}}
	header, ok := dataGridRowTSV(columns, rows, 0)
	if !ok || header != "id\tname" {
		t.Fatalf("expected header TSV, got %q ok=%v", header, ok)
	}
	row, ok := dataGridRowTSV(columns, rows, 1)
	if !ok || row != "1\tAda" {
		t.Fatalf("expected row TSV, got %q ok=%v", row, ok)
	}
}

func TestDataGridNextSelectionDefaultsToFirstDataRow(t *testing.T) {
	row, col, ok := dataGridNextSelection(-1, -1, 5, 3, 1, 0)
	if !ok {
		t.Fatal("expected selection to resolve")
	}
	if row != 1 || col != 0 {
		t.Fatalf("expected default row/col 1/0, got %d/%d", row, col)
	}
}

func TestDataGridNextSelectionClampsWithinBounds(t *testing.T) {
	row, col, ok := dataGridNextSelection(4, 2, 5, 3, 2, 3)
	if !ok {
		t.Fatal("expected selection to resolve")
	}
	if row != 4 || col != 2 {
		t.Fatalf("expected clamped row/col 4/2, got %d/%d", row, col)
	}
	row, col, ok = dataGridNextSelection(0, 0, 5, 3, -2, -4)
	if !ok {
		t.Fatal("expected selection to resolve")
	}
	if row != 0 || col != 0 {
		t.Fatalf("expected clamped row/col 0/0, got %d/%d", row, col)
	}
}

func TestDataGridNextSelectionUsesHeaderWhenNoDataRows(t *testing.T) {
	row, col, ok := dataGridNextSelection(-1, -1, 1, 2, 1, 1)
	if !ok {
		t.Fatal("expected selection to resolve")
	}
	if row != 0 || col != 0 {
		t.Fatalf("expected header selection 0/0, got %d/%d", row, col)
	}
}

func TestDataGridNextSelectionSupportsBoundaryJumps(t *testing.T) {
	row, col, ok := dataGridNextSelection(5, 2, 12, 6, dataGridBoundaryStep, 0)
	if !ok {
		t.Fatal("expected bottom boundary selection")
	}
	if row != 11 || col != 2 {
		t.Fatalf("expected bottom row 11 preserving column 2, got %d/%d", row, col)
	}
	row, col, ok = dataGridNextSelection(5, 2, 12, 6, -dataGridBoundaryStep, 0)
	if !ok {
		t.Fatal("expected top boundary selection")
	}
	if row != 0 || col != 2 {
		t.Fatalf("expected top row 0 preserving column 2, got %d/%d", row, col)
	}
	row, col, ok = dataGridNextSelection(5, 2, 12, 6, 0, dataGridBoundaryStep)
	if !ok {
		t.Fatal("expected row end boundary selection")
	}
	if row != 5 || col != 5 {
		t.Fatalf("expected row end column 5 preserving row 5, got %d/%d", row, col)
	}
	row, col, ok = dataGridNextSelection(5, 2, 12, 6, 0, -dataGridBoundaryStep)
	if !ok {
		t.Fatal("expected row start boundary selection")
	}
	if row != 5 || col != 0 {
		t.Fatalf("expected row start column 0 preserving row 5, got %d/%d", row, col)
	}
}

func TestEstimateDataGridColumnWidthsSamplesTailRows(t *testing.T) {
	columns := []string{"value"}
	rows := make([][]string, 300)
	for index := range rows {
		rows[index] = []string{"x"}
	}
	rows[len(rows)-1][0] = "this tail row should widen the column estimate significantly"
	widths := estimateDataGridColumnWidths(columns, rows)
	if len(widths) != 1 {
		t.Fatalf("expected one width, got %#v", widths)
	}
	if widths[0] <= 120 {
		t.Fatalf("expected tail sample to widen the column, got %f", widths[0])
	}
}

func TestSampledDataGridRowIndexesBoundedAndSpanning(t *testing.T) {
	indexes := sampledDataGridRowIndexes(1000, dataGridWidthSampleMaxRowsDefault)
	if len(indexes) == 0 {
		t.Fatal("expected sampled indexes")
	}
	if len(indexes) > dataGridWidthSampleMaxRowsDefault {
		t.Fatalf("expected at most %d samples, got %d", dataGridWidthSampleMaxRowsDefault, len(indexes))
	}
	if indexes[0] != 0 {
		t.Fatalf("expected first sampled row to be 0, got %d", indexes[0])
	}
	if indexes[len(indexes)-1] != 999 {
		t.Fatalf("expected last sampled row to be 999, got %d", indexes[len(indexes)-1])
	}
	for index := 1; index < len(indexes); index++ {
		if indexes[index] <= indexes[index-1] {
			t.Fatalf("expected strictly increasing sampled indexes, got %#v", indexes)
		}
	}
}

func TestDataGridWidthSampleBudgetAdaptsByDensityAndWidth(t *testing.T) {
	if budget := dataGridWidthSampleBudget(100, 4); budget != dataGridWidthSampleMaxRowsDefault {
		t.Fatalf("expected default sample budget %d, got %d", dataGridWidthSampleMaxRowsDefault, budget)
	}
	if budget := dataGridWidthSampleBudget(dataGridDenseRowThreshold, 4); budget != dataGridWidthSampleMaxRowsDense {
		t.Fatalf("expected dense sample budget %d, got %d", dataGridWidthSampleMaxRowsDense, budget)
	}
	if budget := dataGridWidthSampleBudget(dataGridUltraDenseRowThreshold, 4); budget != dataGridWidthSampleMaxRowsUltraDense {
		t.Fatalf("expected ultra-dense sample budget %d, got %d", dataGridWidthSampleMaxRowsUltraDense, budget)
	}
	if budget := dataGridWidthSampleBudget(100, dataGridWideColumnThreshold); budget != dataGridWidthSampleMaxRowsWide {
		t.Fatalf("expected wide-grid sample budget %d, got %d", dataGridWidthSampleMaxRowsWide, budget)
	}
}

func TestChangedDataGridColumnWidthIndexes(t *testing.T) {
	indexes := changedDataGridColumnWidthIndexes([]float32{120, 200}, []float32{120, 200})
	if len(indexes) != 0 {
		t.Fatalf("expected no changed indexes, got %#v", indexes)
	}
	indexes = changedDataGridColumnWidthIndexes([]float32{120, 200}, []float32{120, 240})
	if len(indexes) != 1 || indexes[0] != 1 {
		t.Fatalf("expected one changed index [1], got %#v", indexes)
	}
	indexes = changedDataGridColumnWidthIndexes([]float32{120, 200}, []float32{120.2, 200.1})
	if len(indexes) != 0 {
		t.Fatalf("expected tolerance to ignore small diffs, got %#v", indexes)
	}
	indexes = changedDataGridColumnWidthIndexes([]float32{120}, []float32{120, 240})
	if len(indexes) != 2 || indexes[0] != 0 || indexes[1] != 1 {
		t.Fatalf("expected full reapply on size mismatch, got %#v", indexes)
	}
}

func BenchmarkEstimateDataGridColumnWidthsLarge(b *testing.B) {
	columns := make([]string, 24)
	for index := range columns {
		columns[index] = fmt.Sprintf("column_%02d", index)
	}
	rows := make([][]string, 10000)
	for rowIndex := range rows {
		row := make([]string, len(columns))
		for colIndex := range columns {
			row[colIndex] = fmt.Sprintf("r%05d-c%02d-value", rowIndex, colIndex)
		}
		rows[rowIndex] = row
	}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		_ = estimateDataGridColumnWidths(columns, rows)
	}
}

func TestResolveDataGridRenderPolicyForDenseRows(t *testing.T) {
	policy := resolveDataGridRenderPolicy([]string{"value"}, make([][]string, dataGridDenseRowThreshold))
	if !policy.HideDivider {
		t.Fatal("expected dense row policy to hide separators")
	}
	if policy.MaxWidth != dataGridWidthMaxDense {
		t.Fatalf("expected dense max width %v, got %f", dataGridWidthMaxDense, policy.MaxWidth)
	}
	if policy.Truncation != fyne.TextTruncateClip {
		t.Fatalf("expected dense truncation clip, got %v", policy.Truncation)
	}
}

func TestResolveDataGridRenderPolicyForUltraDenseRows(t *testing.T) {
	policy := resolveDataGridRenderPolicy([]string{"value"}, make([][]string, dataGridUltraDenseRowThreshold))
	if policy.MaxWidth != dataGridWidthMaxUltraDense {
		t.Fatalf("expected ultra dense max width %v, got %f", dataGridWidthMaxUltraDense, policy.MaxWidth)
	}
}

func TestWithDataGridSamplingStatusHintReplacesExistingHint(t *testing.T) {
	value := withDataGridSamplingStatusHint("sales.csv: query complete | Grid: sampled 20/50 rows", 100, 400, 0, true)
	expected := "sales.csv: query complete | Grid: sampled 100/400 rows, dense mode"
	if value != expected {
		t.Fatalf("expected %q, got %q", expected, value)
	}
}

func TestWithDataGridSamplingStatusHintHeaderWidthMode(t *testing.T) {
	value := withDataGridSamplingStatusHint("sales.csv: query complete", 0, 200, 0, false)
	expected := "sales.csv: query complete | Grid: header-width mode"
	if value != expected {
		t.Fatalf("expected %q, got %q", expected, value)
	}
}

func TestWithDataGridSamplingStatusHintShowsColumnCap(t *testing.T) {
	value := withDataGridSamplingStatusHint("sales.csv: query complete", 80, 200, 12, false)
	expected := "sales.csv: query complete | Grid: sampled 80/200 rows, showing first 128 columns"
	if value != expected {
		t.Fatalf("expected %q, got %q", expected, value)
	}
}

func TestDataGridStatusLineSummarizesSelectionAndDensity(t *testing.T) {
	value := dataGridStatusLine(
		25,
		250,
		8,
		0,
		3,
		2,
		80,
		dataGridRenderPolicy{MaxWidth: dataGridWidthMaxDense, HideDivider: true},
	)
	expected := "Rows: 25 shown of 250 | Columns: 8 visible | Density: dense | Sizing: sampled 80/250 rows | Selection: R3 C3 | Copy: Ctrl/Cmd+C cell, Copy row TSV"
	if value != expected {
		t.Fatalf("expected %q, got %q", expected, value)
	}
}

func TestDataGridStatusLineShowsHeaderSelectionAndHiddenColumns(t *testing.T) {
	value := dataGridStatusLine(
		10,
		10,
		dataGridMaxVisibleColumns,
		12,
		0,
		4,
		0,
		dataGridRenderPolicy{MaxWidth: dataGridWidthMax},
	)
	expected := "Rows: 10 shown | Columns: 128 visible (+12 hidden) | Density: standard | Sizing: headers only | Selection: header C5 | Copy: Ctrl/Cmd+C cell, Copy row TSV"
	if value != expected {
		t.Fatalf("expected %q, got %q", expected, value)
	}
}

func TestDataGridStatusLineHandlesEmptyGrid(t *testing.T) {
	value := dataGridStatusLine(0, 0, 0, 0, -1, -1, 0, dataGridRenderPolicy{})
	if value != "Rows: no grid loaded." {
		t.Fatalf("expected empty grid status, got %q", value)
	}
}

func TestDataRowsTextStatusLine(t *testing.T) {
	if value := dataRowsTextStatusLine(""); value != "Rows: no grid loaded." {
		t.Fatalf("expected empty text status, got %q", value)
	}
	value := dataRowsTextStatusLine("plain rows")
	expected := "Rows: text preview loaded | Grid navigation unavailable for this result."
	if value != expected {
		t.Fatalf("expected text status %q, got %q", expected, value)
	}
}

func TestEstimateDataGridColumnWidthsUltraWideUsesFixedWidths(t *testing.T) {
	columns := make([]string, dataGridUltraWideColumnThreshold)
	for index := range columns {
		columns[index] = fmt.Sprintf("col_%d", index)
	}
	rows := [][]string{{strings.Repeat("very-long-value", 20)}}
	widths := estimateDataGridColumnWidths(columns, rows)
	if len(widths) != len(columns) {
		t.Fatalf("expected %d widths, got %d", len(columns), len(widths))
	}
	expected := float32(dataGridUltraWideColumnWidth)
	if widths[0] != expected {
		t.Fatalf("expected first width %f, got %f", expected, widths[0])
	}
	if _, sampled := estimateDataGridColumnWidthsWithSampling(columns, rows); sampled != 0 {
		t.Fatalf("expected ultra-wide mode to skip row sampling, got %d", sampled)
	}
}

func TestCloneDataRowsDenseUsesShallowCopy(t *testing.T) {
	rows := make([][]string, dataGridDenseRowThreshold)
	for index := range rows {
		rows[index] = []string{fmt.Sprintf("row-%d", index)}
	}
	cloned := cloneDataRows(rows)
	if len(cloned) != len(rows) {
		t.Fatalf("expected cloned length %d, got %d", len(rows), len(cloned))
	}
	if &cloned[0][0] != &rows[0][0] {
		t.Fatal("expected dense clone to reuse inner row slices")
	}
}

func TestLimitDataGridColumnsClipsWideResults(t *testing.T) {
	columns := make([]string, 140)
	for index := range columns {
		columns[index] = fmt.Sprintf("c%d", index)
	}
	rows := [][]string{make([]string, 140)}
	for index := range rows[0] {
		rows[0][index] = fmt.Sprintf("v%d", index)
	}
	visibleColumns, visibleRows, clipped := limitDataGridColumns(columns, rows, dataGridMaxVisibleColumns)
	if len(visibleColumns) != dataGridMaxVisibleColumns {
		t.Fatalf("expected %d visible columns, got %d", dataGridMaxVisibleColumns, len(visibleColumns))
	}
	if len(visibleRows[0]) != dataGridMaxVisibleColumns {
		t.Fatalf("expected %d visible row values, got %d", dataGridMaxVisibleColumns, len(visibleRows[0]))
	}
	if clipped != 12 {
		t.Fatalf("expected 12 clipped columns, got %d", clipped)
	}
}

func TestEstimateDataGridColumnWidthsWithSamplingReportsSampleCount(t *testing.T) {
	columns := []string{"value"}
	rows := make([][]string, 300)
	for index := range rows {
		rows[index] = []string{"x"}
	}
	widths, sampled := estimateDataGridColumnWidthsWithSampling(columns, rows)
	if len(widths) != 1 {
		t.Fatalf("expected one width, got %#v", widths)
	}
	expected := len(sampledDataGridRowIndexes(len(rows), dataGridWidthSampleBudget(len(rows), len(columns))))
	if sampled != expected {
		t.Fatalf("expected sampled count %d, got %d", expected, sampled)
	}
}

func TestIsSingleDataRowsTableObject(t *testing.T) {
	table := widget.NewTable(
		func() (int, int) { return 1, 1 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(widget.TableCellID, fyne.CanvasObject) {},
	)
	if !isSingleDataRowsTableObject([]fyne.CanvasObject{table}, table) {
		t.Fatal("expected single table object match")
	}
	if isSingleDataRowsTableObject(nil, table) {
		t.Fatal("expected nil object list to fail")
	}
	if isSingleDataRowsTableObject([]fyne.CanvasObject{widget.NewLabel("x"), table}, table) {
		t.Fatal("expected multi-object list to fail")
	}
}
