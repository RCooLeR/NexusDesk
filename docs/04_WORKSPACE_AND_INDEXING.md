# Workspace And Indexing

## Goals

NexusDesk should understand a workspace without overwhelming the user, the model, or the studio UI.

Indexing should gather enough structure to make files, documents, datasets, and artifacts searchable and useful across IDE, data studio, analytics studio, document, operations, and artifact surfaces, while avoiding unsafe or noisy content.

It should prefer:

- source code
- documentation
- configuration files
- Markdown and text files
- PDFs and office documents
- spreadsheets and CSV files
- images and screenshots
- SQL files and query outputs
- Dockerfiles and Compose files
- logs selected by the user
- generated artifacts

It should avoid by default:

- dependency directories such as `node_modules`
- build outputs such as `dist`, `build`, `target`, `.next`
- binary blobs without preview support
- large archives
- cache directories
- secrets files
- OS/system files
- hidden folders unless explicitly allowed
- very large logs unless selected or sampled

## Workspace Pipeline

```mermaid
flowchart TD
  Start["Open workspace"] --> Policy["Load workspace policy"]
  Policy --> Scan["Scan root directory"]
  Scan --> Classify["Classify files by path, extension, MIME, and size"]
  Classify --> Filter["Apply ignore rules and safety limits"]
  Filter --> Preview["Build file tree and preview metadata"]
  Preview --> Extract["Extract text, tables, images, and metadata"]
  Extract --> Segment["Create document segments"]
  Segment --> Chunk["Build searchable chunks"]
  Chunk --> Index["Update search indexes"]
  Extract --> Profile["Profile datasets"]
  Profile --> Store["Store dataset schema and summaries"]
  Index --> Ready["Workspace ready for studio navigation, chat, search, and analytics"]
```

## Current Implementation Snapshot

The current app implements the first safe workspace slice:

- `app/internal/workspace/scanner.go` scans an approved workspace root.
- The scanner skips noisy folders, symlinks, listings deeper than 10 levels, and oversized result sets.
- The scanner returns nodes in filesystem tree order so descendants stay grouped under their parent directories.
- The frontend renders indexed nodes as an expandable tree and preserves expanded directories across refreshes.
- `SearchWorkspace` searches safe workspace paths and previewable text content within the same ignore/depth limits.
- `app/internal/workspace/preview.go` reads selected files only through a rooted relative path.
- File previews reject traversal, symlinks, binary or unsupported text encoding content, and oversized previews.
- File creates/updates go through `app/internal/workspace/write.go` with rooted paths, size caps, diff previews, and apply-only-after-preview behavior.
- File deletes go through `app/internal/workspace/delete.go` and reject traversal, metadata paths, directories, and symlinks before frontend confirmation.
- File rename/move goes through `app/internal/workspace/move.go` and rejects traversal, metadata paths, directories, symlinks, same-path moves, and overwrites.
- CSV query export reruns the bounded query through `app/internal/workspace/dataset_query.go` and writes a CSV artifact through the artifact manager.
- CSV chart generation goes through `app/internal/workspace/chart.go` and returns bounded category counts or numeric sums.
- Chat context uses the same rooted preview boundary and sends only selected text content or a bounded pack of pinned previews.
- Workspace open/recent/refresh flows are bound through Wails methods on `app/app.go`.
- Recent workspaces are stored in local JSON config through `app/internal/storage/recent_workspaces.go`.

The app does not yet build persistent chunks, embeddings, or a file watcher. Those remain future indexing work.

Studio implication: every indexed item should eventually be able to answer two questions: which surface should open it, and which actions make sense there.

## File Classification

Every file should be assigned a type and preview mode.

Example classes:

```text
code
text
markdown
json
yaml
sql
spreadsheet
csv
pdf
doc
presentation
image
log
docker
database
archive
binary
unknown
```

Preview modes:

```text
editor
read-only text
image viewer
PDF viewer
spreadsheet table
dataset profile
chart artifact
metadata only
unsupported
```

## Ignore Rules

NexusDesk should combine:

- global ignore rules
- workspace ignore rules
- `.gitignore` rules where useful
- explicit user exclusions
- secret-pattern exclusions

Recommended default ignored paths:

```text
.git/
node_modules/
vendor/
dist/
build/
target/
.next/
.cache/
coverage/
tmp/
*.lock when not useful for analysis
```

Lock files can be useful for dependency analysis, so the app should allow targeted inclusion.

## Extraction Pipeline

### Text And Code

For text and code files:

- detect encoding
- read within size limits
- preserve path, extension, and language
- build chunks by logical boundaries
- prefer heading, function, or block boundaries when possible
- keep line ranges for citations and patch previews

Current implementation:

- scans workspace trees up to 10 levels deep with an 800-node default cap
- previews UTF-8 text/code within a 64 KB default cap
- decodes UTF-8 with BOM, UTF-16 LE/BE with or without BOM, and Windows-1251 Cyrillic text previews
- parses CSV files into bounded table previews with lightweight column profiles from a larger capped CSV sample
- persists first dataset profiles for CSV files and XLSX workbook sheet metadata under `.nexusdesk/datasets/`
- renders common image files as capped inline data URLs
- renders PDF files as capped inline data URLs and extracts simple embedded text by page when available
- extracts basic DOCX body text from `word/document.xml`
- searches workspace path names and previewable text/PDF/DOCX content
- sends selected chat context with a smaller 16 KB cap
- sends selected CSV chat context as a structured column profile plus bounded row sample
- builds bounded multi-file context packs from pinned text, CSV, and extracted-PDF previews
- trims partial UTF-8 characters at truncation boundaries
- shows unsupported state for binary or unsupported text-encoding files
- excludes image and PDF data URLs from text chat context, but allows extracted PDF text as context
- creates Markdown report artifacts under `.nexusdesk/artifacts/` from selected previews
- lists generated Markdown, CSV, and SVG artifacts from `.nexusdesk/artifacts/`
- creates and lists first CSV query export artifacts under `.nexusdesk/artifacts/`
- creates and lists first SVG chart artifacts under `.nexusdesk/artifacts/`
- uses Monaco for read-only text/code previews and text/code edit drafts
- supports safe new file drafts, text/code updates, deletes, and renames/moves through backend file-operation boundaries
- does not yet persist line-aware chunks or citations

### Markdown

For Markdown:

- extract headings
- preserve lists and tables
- keep code blocks separate
- build a heading path for each chunk
- expand generated report flows beyond the starter Markdown artifact

### PDF

For PDFs:

- render in UI with a PDF viewer
- extract simple embedded text per page when possible
- store page numbers for extracted text
- support OCR or vision fallback for scanned PDFs later
- avoid pretending a scanned document has text if extraction fails

### Images

For images:

- preview in UI
- store metadata
- optionally extract OCR text
- use vision model only when the selected model supports it
- reference image files explicitly in model calls

### Spreadsheets

For Excel and CSV:

- inspect workbook/sheet names
- detect headers
- count rows and columns
- sample rows
- render a bounded CSV table preview
- infer column types, missing values, distinct counts, and numeric ranges from a larger capped CSV sample
- persist CSV profiles and XLSX sheet metadata in the workspace
- query CSV rows with a bounded search or `column=value` filter
- expand profiling beyond the current capped sample with richer dataset profiles
- optionally load tables into DuckDB
- never send whole large workbooks directly to the LLM

### Logs

For logs:

- detect timestamp patterns
- sample large logs
- support user-selected ranges
- extract error clusters when useful
- avoid indexing huge logs without limits

### Docker And Config Files

For Dockerfiles, Compose files, env samples, and config files:

- parse as text
- preserve indentation
- tag as operations-related
- highlight dangerous or privileged settings in analysis
- avoid exposing secrets

## Chunking

Chunks should be deterministic and traceable.

Recommended rules:

- keep chunk size below the model context target
- prefer splitting at headings, paragraphs, functions, rows, or pages
- keep overlap only where useful
- attach source metadata: path, line range, page number, sheet, or row range
- discard tiny fragments unless they contain unique metadata
- preserve source hashes

Future improvements:

- AST-aware code chunking
- PDF layout-aware chunking
- spreadsheet semantic regions
- image OCR bounding boxes
- log event clustering
- embeddings for semantic search

## Dataset Profiling

A dataset profile should include:

- source file or connector
- table/sheet name
- row count
- column count
- column names
- inferred types
- missing values
- distinct counts
- numeric min, max, average
- date ranges
- sample rows
- warnings for suspicious data

This gives the LLM a compact, accurate view of the data without loading everything into the prompt.

## Incremental Indexing

Use hashes to avoid reprocessing unchanged files:

- file content hash
- extracted text hash
- chunk hash
- dataset schema hash
- profile hash

When content changes:

- update file metadata
- replace extracted document text
- replace affected chunks
- clear stale summaries
- refresh dataset profile
- refresh search index
- mark related conversations as using stale context if needed

## Workspace Watcher

A file watcher can update the index when files change.

Rules:

- debounce rapid changes
- ignore temporary editor files
- pause indexing during large workspace operations
- show indexing status in the UI
- allow manual reindex

## Index Run Reporting

Every indexing run should record:

- workspace ID
- start/end time
- files discovered
- files indexed
- files skipped
- files failed
- datasets profiled
- chunks created
- artifacts detected
- average extraction latency
- top error types
- ignored path examples

Indexing trust matters. A successful index run is not just "finished"; it should explain what was included and what was skipped.
