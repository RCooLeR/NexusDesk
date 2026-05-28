package artifacts

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const presentationDeckFormat = "pptx"

func BuildPresentationDeckReport(title string, sourcePath string, sourceTitle string, sourceKind string, outlineText string, sourcePaths []string) PresentationDeckReport {
	sourceTitle = strings.TrimSpace(sourceTitle)
	title = strings.TrimSpace(title)
	if title == "" {
		title = presentationDeckTitle(sourceTitle, sourcePath)
	}
	outline := presentationOutlineSlideSection(outlineText)
	slides := presentationSlidesFromMarkdown(outline)
	return PresentationDeckReport{
		Title:       title,
		SourcePath:  filepath.ToSlash(strings.TrimSpace(sourcePath)),
		SourceTitle: sourceTitle,
		SourceKind:  strings.TrimSpace(sourceKind),
		SourcePaths: append([]string{}, sourcePaths...),
		Outline:     outline,
		SlideCount:  len(slides),
		GeneratedBy: "Nexus native PPTX writer",
	}
}

func (s *Store) WritePresentationDeckReport(report PresentationDeckReport) (Artifact, error) {
	outline := strings.TrimSpace(report.Outline)
	if outline == "" {
		return Artifact{}, errors.New("presentation deck outline is required")
	}
	createdAt := time.Now().UTC()
	title := strings.TrimSpace(report.Title)
	if title == "" {
		title = presentationDeckTitle(report.SourceTitle, report.SourcePath)
	}
	relPath := s.relPath("presentation-decks", fmt.Sprintf("%s-%s.pptx", artifactTimestamp(createdAt), safeName(title)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	sourcePaths := presentationOutlineSourcePaths(report.SourcePath, report.SourcePaths)
	slides := presentationSlidesFromMarkdown(outline)
	files := presentationDeckPackageFiles(len(slides))
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	zipWriter := zip.NewWriter(file)
	writeErr := writePresentationDeckZip(zipWriter, report, title, outline, sourcePaths, slides, files, createdAt)
	closeZipErr := zipWriter.Close()
	closeFileErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(absPath)
		return Artifact{}, writeErr
	}
	if closeZipErr != nil {
		_ = os.Remove(absPath)
		return Artifact{}, closeZipErr
	}
	if closeFileErr != nil {
		_ = os.Remove(absPath)
		return Artifact{}, closeFileErr
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return Artifact{}, err
	}
	metadata := Metadata{
		Kind:         "presentation-deck",
		Title:        title,
		RelPath:      relPath,
		Source:       filepath.ToSlash(strings.TrimSpace(report.SourcePath)),
		SourcePaths:  sourcePaths,
		GeneratedAt:  createdAt,
		ExportFormat: presentationDeckFormat,
		PackageFiles: append([]string{}, files...),
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:         "presentation-deck",
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      "Presentation PPTX deck artifact created at " + relPath + ".",
		Size:         info.Size(),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  append([]string{}, sourcePaths...),
	}, nil
}

func writePresentationDeckZip(zipWriter *zip.Writer, report PresentationDeckReport, title string, outline string, sourcePaths []string, slides []presentationSlide, files []string, createdAt time.Time) error {
	if err := addZipText(zipWriter, "[Content_Types].xml", presentationDeckContentTypes(len(slides))); err != nil {
		return err
	}
	if err := addZipText(zipWriter, "_rels/.rels", presentationDeckRootRels()); err != nil {
		return err
	}
	if err := addZipText(zipWriter, "docProps/core.xml", documentExportCoreProperties(DocumentExportReport{GeneratedBy: firstNonEmptyArtifact(report.GeneratedBy, "Nexus native PPTX writer")}, title, createdAt)); err != nil {
		return err
	}
	if err := addZipText(zipWriter, "docProps/app.xml", presentationDeckAppProperties(len(slides))); err != nil {
		return err
	}
	if err := addZipText(zipWriter, "ppt/presentation.xml", presentationDeckPresentationXML(slides)); err != nil {
		return err
	}
	if err := addZipText(zipWriter, "ppt/_rels/presentation.xml.rels", presentationDeckPresentationRels(len(slides))); err != nil {
		return err
	}
	for index, slide := range slides {
		if err := addZipText(zipWriter, fmt.Sprintf("ppt/slides/slide%d.xml", index+1), presentationDeckSlideXML(slide, index+1)); err != nil {
			return err
		}
	}
	return nil
}

func presentationDeckPackageFiles(slideCount int) []string {
	files := []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"docProps/core.xml",
		"docProps/app.xml",
		"ppt/presentation.xml",
		"ppt/_rels/presentation.xml.rels",
	}
	for index := 1; index <= slideCount; index++ {
		files = append(files, fmt.Sprintf("ppt/slides/slide%d.xml", index))
	}
	return files
}

func presentationDeckContentTypes(slideCount int) string {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	builder.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	builder.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	builder.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	builder.WriteString(`<Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>`)
	builder.WriteString(`<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>`)
	builder.WriteString(`<Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>`)
	for index := 1; index <= slideCount; index++ {
		builder.WriteString(`<Override PartName="/ppt/slides/slide`)
		builder.WriteString(fmt.Sprintf("%d", index))
		builder.WriteString(`.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`)
	}
	builder.WriteString(`</Types>`)
	return builder.String()
}

func presentationDeckRootRels() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/></Relationships>`
}

func presentationDeckAppProperties(slideCount int) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"><Application>NexusDesk</Application><Slides>` + fmt.Sprintf("%d", slideCount) + `</Slides></Properties>`
}

func presentationDeckPresentationXML(slides []presentationSlide) string {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	builder.WriteString(`<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:sldIdLst>`)
	for index := range slides {
		builder.WriteString(`<p:sldId id="`)
		builder.WriteString(fmt.Sprintf("%d", 256+index))
		builder.WriteString(`" r:id="rId`)
		builder.WriteString(fmt.Sprintf("%d", index+1))
		builder.WriteString(`"/>`)
	}
	builder.WriteString(`</p:sldIdLst><p:sldSz cx="12192000" cy="6858000" type="screen16x9"/><p:notesSz cx="6858000" cy="9144000"/></p:presentation>`)
	return builder.String()
}

func presentationDeckPresentationRels(slideCount int) string {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for index := 1; index <= slideCount; index++ {
		builder.WriteString(`<Relationship Id="rId`)
		builder.WriteString(fmt.Sprintf("%d", index))
		builder.WriteString(`" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide`)
		builder.WriteString(fmt.Sprintf("%d", index))
		builder.WriteString(`.xml"/>`)
	}
	builder.WriteString(`</Relationships>`)
	return builder.String()
}

func presentationDeckSlideXML(slide presentationSlide, slideNumber int) string {
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	builder.WriteString(`<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree>`)
	builder.WriteString(`<p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/><a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>`)
	builder.WriteString(presentationDeckTextShape(2, "Title", 609600, 457200, 10972800, 914400, slide.Title, nil))
	builder.WriteString(presentationDeckTextShape(3, "Body", 914400, 1676400, 10363200, 4267200, "", slide.Bullets))
	builder.WriteString(`</p:spTree></p:cSld></p:sld>`)
	return builder.String()
}

func presentationDeckTextShape(id int, name string, x int, y int, cx int, cy int, title string, bullets []string) string {
	var builder strings.Builder
	builder.WriteString(`<p:sp><p:nvSpPr><p:cNvPr id="`)
	builder.WriteString(fmt.Sprintf("%d", id))
	builder.WriteString(`" name="`)
	builder.WriteString(xmlEscape(name))
	builder.WriteString(` `)
	builder.WriteString(fmt.Sprintf("%d", id))
	builder.WriteString(`"/><p:cNvSpPr txBox="1"/><p:nvPr/></p:nvSpPr><p:spPr><a:xfrm><a:off x="`)
	builder.WriteString(fmt.Sprintf("%d", x))
	builder.WriteString(`" y="`)
	builder.WriteString(fmt.Sprintf("%d", y))
	builder.WriteString(`"/><a:ext cx="`)
	builder.WriteString(fmt.Sprintf("%d", cx))
	builder.WriteString(`" cy="`)
	builder.WriteString(fmt.Sprintf("%d", cy))
	builder.WriteString(`"/></a:xfrm></p:spPr><p:txBody><a:bodyPr wrap="square"/><a:lstStyle/>`)
	if strings.TrimSpace(title) != "" {
		builder.WriteString(`<a:p><a:r><a:rPr lang="en-US" sz="3600" b="1"/><a:t>`)
		builder.WriteString(xmlEscape(title))
		builder.WriteString(`</a:t></a:r></a:p>`)
	}
	for _, bullet := range bullets {
		builder.WriteString(`<a:p><a:pPr marL="342900" indent="-171450"/><a:r><a:rPr lang="en-US" sz="2200"/><a:t>`)
		builder.WriteString(xmlEscape(bullet))
		builder.WriteString(`</a:t></a:r></a:p>`)
	}
	builder.WriteString(`</p:txBody></p:sp>`)
	return builder.String()
}

func presentationDeckTitle(sourceTitle string, sourcePath string) string {
	sourceTitle = strings.TrimSpace(sourceTitle)
	for _, prefix := range []string{"Presentation Deck - ", "Presentation Package - ", "Presentation Outline - "} {
		sourceTitle = strings.TrimPrefix(sourceTitle, prefix)
	}
	if sourceTitle != "" {
		return "Presentation Deck - " + sourceTitle
	}
	sourcePath = filepath.ToSlash(strings.TrimSpace(sourcePath))
	if sourcePath != "" {
		name := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
		if strings.TrimSpace(name) != "" {
			return "Presentation Deck - " + name
		}
	}
	return "Presentation Deck"
}
