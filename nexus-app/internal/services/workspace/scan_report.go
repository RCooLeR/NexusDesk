package workspace

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
)

const (
	defaultScanReportMaxDepth   = 12
	defaultScanReportMaxEntries = 5000
	defaultScanReportMaxSamples = 12
)

type ScanReportOptions struct {
	MaxDepth   int
	MaxEntries int
	MaxSamples int
}

type ScanReport struct {
	Name           string
	Root           string
	Included       int
	Ignored        int
	DepthSkipped   int
	EntrySkipped   int
	Unreadable     int
	MaxDepth       int
	MaxEntries     int
	Truncated      bool
	IgnoredSamples []string
	SkippedSamples []string
}

func (s *Service) ScanReport(ctx context.Context, root string, options ScanReportOptions) (ScanReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	absRoot, err := cleanRoot(root)
	if err != nil {
		return ScanReport{}, err
	}
	report := ScanReport{
		Name:       filepath.Base(absRoot),
		Root:       absRoot,
		MaxDepth:   normalizedScanReportLimit(options.MaxDepth, defaultScanReportMaxDepth),
		MaxEntries: normalizedScanReportLimit(options.MaxEntries, defaultScanReportMaxEntries),
	}
	maxSamples := normalizedScanReportLimit(options.MaxSamples, defaultScanReportMaxSamples)
	err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		relPath, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			return relErr
		}
		relPath = filepath.ToSlash(relPath)
		if relPath == "." {
			return nil
		}
		if walkErr != nil {
			report.Unreadable++
			report.Truncated = true
			appendScanReportSample(&report.SkippedSamples, maxSamples, "unreadable: "+relPath)
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if depthOf(relPath) > report.MaxDepth {
			report.DepthSkipped++
			report.Truncated = true
			appendScanReportSample(&report.SkippedSamples, maxSamples, "depth: "+relPath)
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if isIgnoredName(entry.Name()) || isInternalMetadataPath(relPath) {
			report.Ignored++
			appendScanReportSample(&report.IgnoredSamples, maxSamples, "ignored: "+relPath)
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			report.Unreadable++
			report.Truncated = true
			appendScanReportSample(&report.SkippedSamples, maxSamples, "unreadable: "+relPath)
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			report.Ignored++
			appendScanReportSample(&report.IgnoredSamples, maxSamples, "symlink: "+relPath)
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if report.Included >= report.MaxEntries {
			report.EntrySkipped++
			report.Truncated = true
			appendScanReportSample(&report.SkippedSamples, maxSamples, "entry cap: "+relPath)
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		report.Included++
		return nil
	})
	if err != nil {
		return ScanReport{}, err
	}
	return report, nil
}

func normalizedScanReportLimit(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func appendScanReportSample(samples *[]string, maxSamples int, sample string) {
	if maxSamples <= 0 || len(*samples) >= maxSamples {
		return
	}
	*samples = append(*samples, sample)
}

func (r ScanReport) Message() string {
	skipped := r.Ignored + r.DepthSkipped + r.EntrySkipped + r.Unreadable
	return fmt.Sprintf("Scanned %d workspace entries, skipped %d.", r.Included, skipped)
}
