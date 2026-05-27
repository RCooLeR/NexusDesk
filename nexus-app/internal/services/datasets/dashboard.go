package datasets

import (
	"errors"
	"fmt"
	"html"
	"math"
	"strings"
)

func BuildDashboard(result QueryResult) (DashboardResult, error) {
	if len(result.Columns) == 0 || len(result.Rows) == 0 {
		return DashboardResult{}, errors.New("dashboard requires a bounded query result with columns and rows")
	}
	chart, err := BuildChart(result)
	if err != nil {
		return DashboardResult{}, err
	}
	dashboard := DashboardResult{
		RelPath:   result.RelPath,
		Query:     result.Query,
		Format:    result.Format,
		Metrics:   dashboardMetrics(result, chart),
		Chart:     chart,
		Truncated: result.Truncated || chart.Truncated,
	}
	dashboard.Message = fmt.Sprintf("Dashboard: %d metric(s), %s", len(dashboard.Metrics), chart.Message)
	dashboard.SVG = dashboardSVG(dashboard)
	return dashboard, nil
}

func dashboardMetrics(result QueryResult, chart ChartResult) []DashboardMetric {
	metrics := []DashboardMetric{
		{Label: "Shown rows", Value: fmt.Sprintf("%d", len(result.Rows)), Detail: fmt.Sprintf("%d matched", result.MatchedRows)},
		{Label: "Columns", Value: fmt.Sprintf("%d", len(result.Columns)), Detail: result.Format},
	}
	if result.TotalRows > 0 {
		metrics = append(metrics, DashboardMetric{Label: "Loaded rows", Value: fmt.Sprintf("%d", result.TotalRows), Detail: "bounded source sample"})
	}
	if chart.ValueColumn != "" && len(chart.Points) > 0 {
		total := 0.0
		for _, point := range chart.Points {
			total += point.Value
		}
		label := "Total " + chart.ValueColumn
		if chart.Mode == "line" {
			label = "Latest " + chart.ValueColumn
			total = chart.Points[len(chart.Points)-1].Value
		}
		metrics = append(metrics, DashboardMetric{Label: label, Value: formatChartValue(total), Detail: chart.CategoryColumn})
	}
	numericCount := numericColumnCount(result.Rows)
	if numericCount > 0 {
		metrics = append(metrics, DashboardMetric{Label: "Numeric fields", Value: fmt.Sprintf("%d", numericCount), Detail: "detected in sample"})
	}
	return metrics
}

func numericColumnCount(rows [][]string) int {
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	count := 0
	for index := 0; index < width; index++ {
		for _, row := range rows {
			if _, ok := parseNumber(valueAt(row, index)); ok {
				count++
				break
			}
		}
	}
	return count
}

func dashboardSVG(dashboard DashboardResult) string {
	const (
		width       = 1040
		height      = 680
		cardTop     = 84
		cardWidth   = 220
		cardHeight  = 88
		cardGap     = 18
		chartLeft   = 48
		chartTop    = 218
		chartWidth  = 650
		chartHeight = 350
		sideLeft    = 730
		sideTop     = 218
	)
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, width, height, width, height))
	builder.WriteString(`<rect width="100%" height="100%" fill="#f8fafc"/>`)
	builder.WriteString(`<rect x="0" y="0" width="1040" height="64" fill="#111827"/>`)
	builder.WriteString(fmt.Sprintf(`<text x="48" y="40" font-family="Segoe UI, Arial, sans-serif" font-size="22" font-weight="700" fill="#f9fafb">%s</text>`, html.EscapeString("Dataset Dashboard")))
	builder.WriteString(fmt.Sprintf(`<text x="296" y="40" font-family="Segoe UI, Arial, sans-serif" font-size="13" fill="#cbd5e1">%s</text>`, html.EscapeString(compactChartLabel(dashboard.RelPath))))

	for index, metric := range dashboard.Metrics {
		if index >= 4 {
			break
		}
		x := 48 + index*(cardWidth+cardGap)
		builder.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="8" fill="#ffffff" stroke="#dbe3ea"/>`, x, cardTop, cardWidth, cardHeight))
		builder.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Segoe UI, Arial, sans-serif" font-size="12" fill="#64748b">%s</text>`, x+18, cardTop+26, html.EscapeString(metric.Label)))
		builder.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Segoe UI, Arial, sans-serif" font-size="28" font-weight="700" fill="#111827">%s</text>`, x+18, cardTop+58, html.EscapeString(metric.Value)))
		if strings.TrimSpace(metric.Detail) != "" {
			builder.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Segoe UI, Arial, sans-serif" font-size="11" fill="#64748b">%s</text>`, x+18, cardTop+76, html.EscapeString(metric.Detail)))
		}
	}

	builder.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="8" fill="#ffffff" stroke="#dbe3ea"/>`, chartLeft, chartTop, chartWidth, chartHeight))
	builder.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Segoe UI, Arial, sans-serif" font-size="16" font-weight="700" fill="#111827">%s</text>`, chartLeft+22, chartTop+34, html.EscapeString(chartTitle(dashboard.Chart))))
	drawDashboardChart(&builder, dashboard.Chart, chartLeft+54, chartTop+62, chartWidth-92, chartHeight-116)

	builder.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="262" height="350" rx="8" fill="#ffffff" stroke="#dbe3ea"/>`, sideLeft, sideTop))
	builder.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Segoe UI, Arial, sans-serif" font-size="16" font-weight="700" fill="#111827">Dataset Notes</text>`, sideLeft+22, sideTop+34))
	sideLines := []string{
		"Format: " + dashboard.Format,
		"Chart mode: " + dashboard.Chart.Mode,
		"Category: " + dashboard.Chart.CategoryColumn,
		"Value: " + firstNonEmptyDashboard(dashboard.Chart.ValueColumn, "row count"),
		fmt.Sprintf("Points: %d", len(dashboard.Chart.Points)),
	}
	if strings.TrimSpace(dashboard.Query) != "" {
		sideLines = append(sideLines, "Query: "+dashboard.Query)
	}
	if dashboard.Truncated {
		sideLines = append(sideLines, "Scope: bounded sample")
	}
	for index, line := range sideLines {
		builder.WriteString(fmt.Sprintf(`<text x="%d" y="%d" font-family="Segoe UI, Arial, sans-serif" font-size="12" fill="#334155">%s</text>`, sideLeft+22, sideTop+68+index*26, html.EscapeString(compactDashboardLine(line, 36))))
	}
	builder.WriteString(fmt.Sprintf(`<text x="48" y="626" font-family="Segoe UI, Arial, sans-serif" font-size="12" fill="#475569">%s</text>`, html.EscapeString(dashboard.Message)))
	builder.WriteString(`</svg>`)
	return builder.String()
}

func drawDashboardChart(builder *strings.Builder, chart ChartResult, left int, top int, plotWidth int, plotHeight int) {
	maxValue, minValue := chartExtents(chart.Points)
	builder.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#94a3b8" stroke-width="1"/>`, left, top+plotHeight, left+plotWidth, top+plotHeight))
	builder.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#94a3b8" stroke-width="1"/>`, left, top, left, top+plotHeight))
	for grid := 1; grid <= 3; grid++ {
		y := float64(top) + float64(plotHeight)*float64(grid)/4
		builder.WriteString(fmt.Sprintf(`<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="#e2e8f0" stroke-width="1"/>`, left, y, left+plotWidth, y))
	}
	if chart.Mode == "line" {
		writeLineChartSVG(builder, chart, left, top, plotWidth, plotHeight, minValue, maxValue)
		return
	}
	if maxValue <= 0 {
		maxValue = 1
	}
	barGap := 8.0
	barWidth := (float64(plotWidth) - barGap*float64(max(0, len(chart.Points)-1))) / float64(max(1, len(chart.Points)))
	if barWidth < 8 {
		barWidth = 8
	}
	writeBarChartSVG(builder, chart, left, top, plotWidth, plotHeight, maxValue, barWidth, barGap)
}

func chartExtents(points []ChartPoint) (float64, float64) {
	if len(points) == 0 {
		return 1, 0
	}
	maxValue := points[0].Value
	minValue := points[0].Value
	for _, point := range points {
		maxValue = math.Max(maxValue, point.Value)
		minValue = math.Min(minValue, point.Value)
	}
	return maxValue, minValue
}

func compactDashboardLine(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func firstNonEmptyDashboard(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
