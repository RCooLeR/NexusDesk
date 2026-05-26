package workspace

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

const datasetChartMaxPoints = 12

type DatasetChartRequest struct {
	RelPath        string `json:"relPath"`
	ChartType      string `json:"chartType"`
	CategoryColumn string `json:"categoryColumn"`
	ValueColumn    string `json:"valueColumn"`
}

type DatasetChartPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Count int     `json:"count"`
}

type DatasetChartResult struct {
	RelPath        string              `json:"relPath"`
	ChartType      string              `json:"chartType"`
	CategoryColumn string              `json:"categoryColumn"`
	ValueColumn    string              `json:"valueColumn"`
	Mode           string              `json:"mode"`
	Points         []DatasetChartPoint `json:"points"`
	TotalRows      int                 `json:"totalRows"`
	UsedRows       int                 `json:"usedRows"`
	Message        string              `json:"message"`
}

func BuildCSVChart(root string, request DatasetChartRequest) (DatasetChartResult, error) {
	preview, err := Preview(root, request.RelPath, PreviewOptions{MaxBytes: csvProfileMaxBytes})
	if err != nil {
		return DatasetChartResult{}, err
	}
	columns, records, err := queryableDatasetRows(preview)
	if err != nil {
		return DatasetChartResult{}, err
	}
	if len(records) == 0 {
		return DatasetChartResult{}, errors.New("dataset needs at least one data row")
	}

	categoryIndex := columnIndexByName(columns, request.CategoryColumn)
	if categoryIndex < 0 {
		return DatasetChartResult{}, errors.New("choose a category column for the chart")
	}

	valueIndex := columnIndexByName(columns, request.ValueColumn)
	mode := "count"
	if valueIndex >= 0 {
		mode = "sum"
	}

	buckets := map[string]*DatasetChartPoint{}
	labels := []string{}
	totalRows := 0
	usedRows := 0
	for _, record := range records {
		totalRows++
		if categoryIndex >= len(record) {
			continue
		}

		label := strings.TrimSpace(record[categoryIndex])
		if label == "" {
			label = "(blank)"
		}

		value := 1.0
		if valueIndex >= 0 {
			if valueIndex >= len(record) {
				continue
			}
			parsed, ok := parseChartNumber(record[valueIndex])
			if !ok {
				continue
			}
			value = parsed
		}

		point := buckets[label]
		if point == nil {
			point = &DatasetChartPoint{Label: label}
			buckets[label] = point
			labels = append(labels, label)
		}
		point.Value += value
		point.Count++
		usedRows++
	}

	if len(buckets) == 0 {
		return DatasetChartResult{}, errors.New("no chartable rows found for the selected columns")
	}

	points := make([]DatasetChartPoint, 0, len(labels))
	for _, label := range labels {
		point := buckets[label]
		if math.IsNaN(point.Value) || math.IsInf(point.Value, 0) {
			continue
		}
		points = append(points, *point)
	}

	chartType := strings.ToLower(strings.TrimSpace(request.ChartType))
	if chartType == "" {
		chartType = "bar"
	}
	if chartType != "bar" && chartType != "line" {
		return DatasetChartResult{}, errors.New("chart type must be bar or line")
	}
	if chartType == "bar" {
		sort.SliceStable(points, func(i, j int) bool {
			if points[i].Value == points[j].Value {
				return strings.ToLower(points[i].Label) < strings.ToLower(points[j].Label)
			}
			return points[i].Value > points[j].Value
		})
	}
	if len(points) > datasetChartMaxPoints {
		points = points[:datasetChartMaxPoints]
	}

	message := fmt.Sprintf("Charted %d categories from %s.", len(points), preview.RelPath)
	if mode == "sum" {
		message = fmt.Sprintf("Charted %d %s totals by %s from %s.", len(points), columns[valueIndex], columns[categoryIndex], preview.RelPath)
	}

	return DatasetChartResult{
		RelPath:        preview.RelPath,
		ChartType:      chartType,
		CategoryColumn: columns[categoryIndex],
		ValueColumn:    selectedColumnName(columns, valueIndex),
		Mode:           mode,
		Points:         points,
		TotalRows:      totalRows,
		UsedRows:       usedRows,
		Message:        message,
	}, nil
}

func columnIndexByName(columns []string, name string) int {
	name = strings.TrimSpace(name)
	if name == "" {
		return -1
	}
	for index, column := range columns {
		if strings.EqualFold(strings.TrimSpace(column), name) {
			return index
		}
	}
	return -1
}

func selectedColumnName(columns []string, index int) string {
	if index < 0 || index >= len(columns) {
		return ""
	}
	return columns[index]
}

func parseChartNumber(value string) (float64, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if value == "" {
		return 0, false
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return number, true
}
