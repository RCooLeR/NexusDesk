package datasets

import (
	"errors"
	"fmt"
	"html"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const chartMaxPoints = 24

func BuildChart(result QueryResult) (ChartResult, error) {
	if len(result.Columns) == 0 || len(result.Rows) == 0 {
		return ChartResult{}, errors.New("chart requires a bounded query result with columns and rows")
	}
	categoryIndex := 0
	valueIndex := firstNumericColumn(result.Rows, categoryIndex)
	if valueIndex >= 0 {
		points, ok := lineChartPoints(result, categoryIndex, valueIndex)
		if ok {
			truncated := false
			if len(points) > chartMaxPoints {
				points = points[:chartMaxPoints]
				truncated = true
			}
			chart := ChartResult{
				RelPath:        result.RelPath,
				Query:          result.Query,
				Format:         result.Format,
				Mode:           "line",
				CategoryColumn: result.Columns[categoryIndex],
				ValueColumn:    result.Columns[valueIndex],
				Points:         points,
				Truncated:      truncated || result.Truncated,
				Message:        chartMessage(points, "line", result.Columns[categoryIndex], result.Columns[valueIndex], truncated || result.Truncated),
			}
			chart.SVG = chartSVG(chart)
			return chart, nil
		}
	}
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

func lineChartPoints(result QueryResult, categoryIndex int, valueIndex int) ([]ChartPoint, bool) {
	type orderedPoint struct {
		point ChartPoint
		order float64
	}
	points := make([]orderedPoint, 0, len(result.Rows))
	for _, row := range result.Rows {
		label := strings.TrimSpace(valueAt(row, categoryIndex))
		value, valueOK := parseNumber(valueAt(row, valueIndex))
		order, orderOK := parseChartOrder(label)
		if label == "" || !valueOK || !orderOK {
			return nil, false
		}
		points = append(points, orderedPoint{
			point: ChartPoint{Label: label, Value: value},
			order: order,
		})
	}
	if len(points) < 2 {
		return nil, false
	}
	sort.SliceStable(points, func(i, j int) bool {
		return points[i].order < points[j].order
	})
	resultPoints := make([]ChartPoint, 0, len(points))
	for _, point := range points {
		resultPoints = append(resultPoints, point.point)
	}
	return resultPoints, true
}

func parseChartOrder(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if number, ok := parseNumber(value); ok {
		return number, true
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
		"02.01.2006",
		"2006-01",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return float64(parsed.Unix()), true
		}
	}
	return 0, false
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
	if mode == "line" {
		return fmt.Sprintf("Line chart: %s over %s across %d ordered points%s.", valueColumn, categoryColumn, len(points), chartTruncatedSuffix(truncated))
	}
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
	minValue := 0.0
	if len(chart.Points) > 0 {
		maxValue = chart.Points[0].Value
		minValue = chart.Points[0].Value
		for _, point := range chart.Points {
			if point.Value > maxValue {
				maxValue = point.Value
			}
			if point.Value < minValue {
				minValue = point.Value
			}
		}
	}
	if chart.Mode != "line" && maxValue <= 0 {
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
	for grid := 1; grid <= 3; grid++ {
		y := float64(topMargin) + float64(plotHeight)*float64(grid)/4
		builder.WriteString(fmt.Sprintf(`<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="#e2e8f0" stroke-width="1"/>`, leftMargin, y, leftMargin+plotWidth, y))
	}
	if chart.Mode == "line" {
		writeLineChartSVG(&builder, chart, leftMargin, topMargin, plotWidth, plotHeight, minValue, maxValue)
	} else {
		writeBarChartSVG(&builder, chart, leftMargin, topMargin, plotWidth, plotHeight, maxValue, barWidth, barGap)
	}
	builder.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Segoe UI, Arial, sans-serif" font-size="12" fill="#475569">%s</text>`, leftMargin, height-16, html.EscapeString(chart.Message)))
	builder.WriteString(`</svg>`)
	return builder.String()
}

func writeBarChartSVG(builder *strings.Builder, chart ChartResult, leftMargin int, topMargin int, plotWidth int, plotHeight int, maxValue float64, barWidth float64, barGap float64) {
	for index, point := range chart.Points {
		x := float64(leftMargin) + float64(index)*(barWidth+barGap)
		barHeight := (point.Value / maxValue) * float64(plotHeight)
		y := float64(topMargin+plotHeight) - barHeight
		builder.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="3" fill="#2563eb"/>`, x, y, barWidth, barHeight))
		builder.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" font-family="Segoe UI, Arial, sans-serif" font-size="11" fill="#111827" text-anchor="middle">%s</text>`, x+barWidth/2, y-6, html.EscapeString(formatChartValue(point.Value))))
		builder.WriteString(fmt.Sprintf(`<text transform="translate(%.1f %.1f) rotate(-35)" font-family="Segoe UI, Arial, sans-serif" font-size="11" fill="#334155" text-anchor="end">%s</text>`, x+barWidth/2, float64(topMargin+plotHeight+24), html.EscapeString(compactChartLabel(point.Label))))
	}
}

func writeLineChartSVG(builder *strings.Builder, chart ChartResult, leftMargin int, topMargin int, plotWidth int, plotHeight int, minValue float64, maxValue float64) {
	if len(chart.Points) == 0 {
		return
	}
	valueRange := maxValue - minValue
	if math.Abs(valueRange) < 0.000001 {
		valueRange = 1
	}
	step := 0.0
	if len(chart.Points) > 1 {
		step = float64(plotWidth) / float64(len(chart.Points)-1)
	}
	coordinates := make([]string, 0, len(chart.Points))
	for index, point := range chart.Points {
		x := float64(leftMargin) + float64(index)*step
		y := float64(topMargin+plotHeight) - ((point.Value-minValue)/valueRange)*float64(plotHeight)
		coordinates = append(coordinates, fmt.Sprintf("%.1f,%.1f", x, y))
	}
	builder.WriteString(fmt.Sprintf(`<polyline points="%s" fill="none" stroke="#2563eb" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>`, strings.Join(coordinates, " ")))
	for index, point := range chart.Points {
		x := float64(leftMargin) + float64(index)*step
		y := float64(topMargin+plotHeight) - ((point.Value-minValue)/valueRange)*float64(plotHeight)
		builder.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="4" fill="#f8fafc" stroke="#2563eb" stroke-width="2"/>`, x, y))
		if index == 0 || index == len(chart.Points)-1 || len(chart.Points) <= 8 {
			builder.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" font-family="Segoe UI, Arial, sans-serif" font-size="11" fill="#111827" text-anchor="middle">%s</text>`, x, y-10, html.EscapeString(formatChartValue(point.Value))))
		}
		if index == 0 || index == len(chart.Points)-1 || index%max(1, len(chart.Points)/6) == 0 {
			builder.WriteString(fmt.Sprintf(`<text transform="translate(%.1f %.1f) rotate(-35)" font-family="Segoe UI, Arial, sans-serif" font-size="11" fill="#334155" text-anchor="end">%s</text>`, x, float64(topMargin+plotHeight+24), html.EscapeString(compactChartLabel(point.Label))))
		}
	}
}

func chartTitle(chart ChartResult) string {
	if chart.Mode == "line" {
		return fmt.Sprintf("%s over %s", chart.ValueColumn, chart.CategoryColumn)
	}
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
