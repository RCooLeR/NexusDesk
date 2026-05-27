package datasets

import (
	"errors"
	"fmt"
	"html"
	"math"
	"sort"
	"strconv"
	"strings"
)

const chartMaxPoints = 24

func BuildChart(result QueryResult) (ChartResult, error) {
	if len(result.Columns) == 0 || len(result.Rows) == 0 {
		return ChartResult{}, errors.New("chart requires a bounded query result with columns and rows")
	}
	categoryIndex := 0
	valueIndex := firstNumericColumn(result.Rows, categoryIndex)
	points, mode, valueColumn := chartPoints(result, categoryIndex, valueIndex)
	if len(points) == 0 {
		return ChartResult{}, errors.New("chart did not find usable values")
	}
	sort.SliceStable(points, func(i, j int) bool {
		if points[i].Value == points[j].Value {
			return points[i].Label < points[j].Label
		}
		return points[i].Value > points[j].Value
	})
	truncated := false
	if len(points) > chartMaxPoints {
		points = points[:chartMaxPoints]
		truncated = true
	}
	chart := ChartResult{
		RelPath:        result.RelPath,
		Query:          result.Query,
		Format:         result.Format,
		Mode:           mode,
		CategoryColumn: result.Columns[categoryIndex],
		ValueColumn:    valueColumn,
		Points:         points,
		Truncated:      truncated || result.Truncated,
		Message:        chartMessage(points, mode, result.Columns[categoryIndex], valueColumn, truncated || result.Truncated),
	}
	chart.SVG = chartSVG(chart)
	return chart, nil
}

func firstNumericColumn(rows [][]string, skip int) int {
	bestIndex := -1
	bestCount := 0
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	for index := 0; index < width; index++ {
		if index == skip {
			continue
		}
		count := 0
		for _, row := range rows {
			if _, ok := parseNumber(valueAt(row, index)); ok {
				count++
			}
		}
		if count > bestCount {
			bestIndex = index
			bestCount = count
		}
	}
	return bestIndex
}

func chartPoints(result QueryResult, categoryIndex int, valueIndex int) ([]ChartPoint, string, string) {
	values := map[string]float64{}
	for _, row := range result.Rows {
		label := strings.TrimSpace(valueAt(row, categoryIndex))
		if label == "" {
			label = "(blank)"
		}
		if valueIndex >= 0 {
			value, ok := parseNumber(valueAt(row, valueIndex))
			if !ok {
				continue
			}
			values[label] += value
			continue
		}
		values[label]++
	}
	points := make([]ChartPoint, 0, len(values))
	for label, value := range values {
		points = append(points, ChartPoint{Label: label, Value: value})
	}
	if valueIndex >= 0 && valueIndex < len(result.Columns) {
		return points, "sum", result.Columns[valueIndex]
	}
	return points, "count", ""
}

func parseNumber(value string) (float64, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if value == "" {
		return 0, false
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false
	}
	return number, true
}

func chartMessage(points []ChartPoint, mode string, categoryColumn string, valueColumn string, truncated bool) string {
	if mode == "sum" {
		return fmt.Sprintf("Bar chart: %s by %s across %d category values%s.", valueColumn, categoryColumn, len(points), chartTruncatedSuffix(truncated))
	}
	return fmt.Sprintf("Bar chart: row counts by %s across %d category values%s.", categoryColumn, len(points), chartTruncatedSuffix(truncated))
}

func chartTruncatedSuffix(truncated bool) string {
	if truncated {
		return " (bounded sample)"
	}
	return ""
}

func chartSVG(chart ChartResult) string {
	const (
		width       = 760
		height      = 420
		leftMargin  = 88
		rightMargin = 28
		topMargin   = 52
		bottom      = 92
	)
	plotWidth := width - leftMargin - rightMargin
	plotHeight := height - topMargin - bottom
	maxValue := 0.0
	for _, point := range chart.Points {
		if point.Value > maxValue {
			maxValue = point.Value
		}
	}
	if maxValue <= 0 {
		maxValue = 1
	}
	barGap := 8.0
	barWidth := (float64(plotWidth) - barGap*float64(max(0, len(chart.Points)-1))) / float64(max(1, len(chart.Points)))
	if barWidth < 8 {
		barWidth = 8
	}
	title := chartTitle(chart)
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, width, height, width, height))
	builder.WriteString(`<rect width="100%" height="100%" fill="#f8fafc"/>`)
	builder.WriteString(fmt.Sprintf(`<text x="%d" y="30" font-family="Segoe UI, Arial, sans-serif" font-size="18" font-weight="700" fill="#111827">%s</text>`, leftMargin, html.EscapeString(title)))
	builder.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#94a3b8" stroke-width="1"/>`, leftMargin, topMargin+plotHeight, leftMargin+plotWidth, topMargin+plotHeight))
	builder.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#94a3b8" stroke-width="1"/>`, leftMargin, topMargin, leftMargin, topMargin+plotHeight))
	for index, point := range chart.Points {
		x := float64(leftMargin) + float64(index)*(barWidth+barGap)
		barHeight := (point.Value / maxValue) * float64(plotHeight)
		y := float64(topMargin+plotHeight) - barHeight
		builder.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="3" fill="#2563eb"/>`, x, y, barWidth, barHeight))
		builder.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" font-family="Segoe UI, Arial, sans-serif" font-size="11" fill="#111827" text-anchor="middle">%s</text>`, x+barWidth/2, y-6, html.EscapeString(formatChartValue(point.Value))))
		builder.WriteString(fmt.Sprintf(`<text transform="translate(%.1f %.1f) rotate(-35)" font-family="Segoe UI, Arial, sans-serif" font-size="11" fill="#334155" text-anchor="end">%s</text>`, x+barWidth/2, float64(topMargin+plotHeight+24), html.EscapeString(compactChartLabel(point.Label))))
	}
	builder.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Segoe UI, Arial, sans-serif" font-size="12" fill="#475569">%s</text>`, leftMargin, height-16, html.EscapeString(chart.Message)))
	builder.WriteString(`</svg>`)
	return builder.String()
}

func chartTitle(chart ChartResult) string {
	if chart.Mode == "sum" {
		return fmt.Sprintf("%s by %s", chart.ValueColumn, chart.CategoryColumn)
	}
	return fmt.Sprintf("Rows by %s", chart.CategoryColumn)
}

func compactChartLabel(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 28 {
		return value[:25] + "..."
	}
	return value
}

func formatChartValue(value float64) string {
	if math.Abs(value-math.Round(value)) < 0.000001 {
		return strconv.FormatInt(int64(math.Round(value)), 10)
	}
	return strconv.FormatFloat(value, 'f', 2, 64)
}
