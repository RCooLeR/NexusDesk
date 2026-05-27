# Workspace And Indexing

## Goals

Nexus Augentic Studio should understand a workspace without overwhelming the user, the model, or the studio UI.

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
- `SearchWorkspace` searches safe workspace paths and previewable text content within the same ignore/depth limits, while `SearchWorkspaceAdvanced` adds regex and lightweight symbol matching for the Workbench search surface.
- `ListWorkspaceProblems` runs a read-only lightweight problem scan over bounded previews for TODO/FIXME/HACK/BUG markers, merge-conflict markers, and invalid JSON.
- `ListWorkspaceTasks` discovers runnable task candidates, and `RunWorkspaceTask` can run only a re-discovered task ID with captured output and a saved task-run artifact.
- `app/internal/workspace/preview.go` reads selected files only through a rooted relative path.
- File previews reject traversal, symlinks, binary or unsupported text encoding content, and oversized previews.
- File creates/updates go through `app/internal/workspace/write.go` with rooted paths, size caps, diff previews, and apply-only-after-preview behavior.
- File deletes go through `app/internal/workspace/delete.go` and reject traversal, metadata paths, directories, and symlinks before frontend confirmation.
- File rename/move goes through `app/internal/workspace/move.go` and rejects traversal, metadata paths, directories, symlinks, same-path moves, and overwrites.
- CSV query export reruns the bounded query through `app/internal/workspace/dataset_query.go` and writes a CSV artifact through the artifact manager.
- Saved CSV row filters and read-only SQL snippets are stored per dataset under `.nexusdesk/datasets/queries.json` as separate query kinds.
- Dataset summary artifacts use the bounded CSV preview/profile data to write deterministic Markdown with column profiles and suggested analysis questions.
- Artifact metadata and chat history are included in the workspace search surface even before a full index database exists.
- SQLite metadata search now adds chat, artifact, and tool-run history snippets once the workspace metadata store exists.
- SQLite workspace files (`.sqlite`, `.sqlite3`, `.db`) are classified as database files and routed to the read-only connector surface rather than text preview.
- `app/internal/workspace/freshness.go` captures file fingerprints so the shell can detect changed files, mark generated artifacts that cite changed sources as stale, and flag dataset-derived views/snippets/reports that should be refreshed.
- Dataset dependency and SQL run records preserve which saved snippets, SQL reports, chart artifacts, query exports, summaries, and connector queries came from a dataset path.
- Chat messages and context-pack previews surface stale-source warnings when their cited files change.
- The workbench can rebuild a context preview from changed files and records that stale-context refresh in the local approval/metadata trail.
- Data & Analytics clears visible query/chart/profile state for the active dataset when that dataset changes on disk.
- Native chart generation goes through `nexus-app/internal/services/datasets/chart.go` and returns bounded categorical bar charts or ordered line charts from query results.
- Chat context uses the same rooted preview boundary and sends only selected text content or a bounded pack of pinned previews.
- Workspace open/recent/refresh/search/read/file mutation/freshness flows keep stable Wails method names on `app/app.go`, but dispatch through `app/workspace_service.go`.
- Recent workspaces are stored in local JSON config through `app/internal/storage/recent_workspaces.go`.

The app does not yet build persistent chunks, embeddings, or an event-driven filesystem watcher. Those remain future indexing work; the current freshness pass is a polling snapshot that keeps stale-source warnings visible without background indexing complexity.

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

Nexus Augentic Studio should combine:

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
- profiles native datasets for CSV, TSV, JSON, NDJSON/JSONL, first-sheet XLSX rows, log lines, and bounded Parquet footer schema/row-group metadata
- renders common image files as capped inline data URLs
- renders PDF files as capped inline data URLs and extracts simple embedded text by page when available
- extracts basic DOCX body text from `word/document.xml`
- searches workspace path names and previewable text/PDF/DOCX content
- sends selected chat context using the active model context-window budget after response reserve and overhead
- sends selected CSV chat context as a structured column profile plus bounded row sample
- builds bounded multi-file context packs from pinned text, CSV, and extracted-PDF previews using the active model context-window budget
- trims partial UTF-8 characters at truncation boundaries
- shows unsupported state for binary or unsupported text-encoding files
- previews and applies safe text writes with explicit UTF-8, UTF-8 BOM, UTF-16 LE/BE, or Windows-1251 output encoding
- excludes image and PDF data URLs from text chat context, but allows extracted PDF text as context
- creates Markdown report artifacts under `.nexusdesk/artifacts/` from selected previews
- lists generated Markdown, CSV, and SVG artifacts from `.nexusdesk/artifacts/`
- creates and lists first CSV query export artifacts under `.nexusdesk/artifacts/`
- creates and lists first SVG chart artifacts under `.nexusdesk/artifacts/`
- uses Monaco for read-only text/code previews and text/code edit drafts
- supports safe new file drafts, text/code updates, deletes, and renames/moves through backend file-operation boundaries
- persists chat/artifact source path citations but does not yet persist line-aware chunks

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
- infer column types, missing values, and sample values from bounded CSV/TSV/JSON/NDJSON/XLSX/log data
- inspect Parquet file size, footer metadata length, schema columns, and row-group byte/row summaries without reading column values
- query CSV, TSV, JSON, NDJSON, XLSX, and log rows with bounded search, `column=value`, comparison, order, and limit filters
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

The current app has a polling freshness check; a later event-driven file watcher can update the index when files change.

Rules:

- debounce rapid changes
- ignore temporary editor files
- pause indexing during large workspace operations
- show indexing status in the UI
- allow manual reindex

Current behavior:

- compares file size and modification time against the last workspace snapshot
- ignores `.git/` and Nexus Augentic Studio metadata/tool-run internals
- marks changed rows in the navigator
- flags generated artifacts whose provenance references changed source files
- warns chat messages and context-pack previews that cited files changed
- invalidates active dataset query, SQL, chart, and profile state for changed selected datasets
- reports dataset-derived refresh needs for changed CSV/XLSX files

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
