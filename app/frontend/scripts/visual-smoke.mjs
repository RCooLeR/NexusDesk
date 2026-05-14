import {existsSync, mkdirSync, readFileSync, statSync, writeFileSync} from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const indexPath = path.join(root, 'dist', 'index.html');
const screenshotDir = path.join(root, 'dist', 'smoke');
const baselineDir = path.join(root, 'visual-baselines');

if (!existsSync(indexPath)) {
    console.error('NexusDesk visual smoke failed: run npm run build first.');
    process.exit(1);
}

let chromium;
try {
    ({chromium} = await import('playwright'));
} catch {
    console.error('NexusDesk visual smoke failed: Playwright is not installed.');
    process.exit(1);
}

mkdirSync(screenshotDir, {recursive: true});
mkdirSync(baselineDir, {recursive: true});
const server = await serveDist(path.join(root, 'dist'));
const browser = await chromium.launch();
const page = await browser.newPage({viewport: {width: 1440, height: 960}});
await page.addInitScript(() => {
    const snapshot = {
        root: 'E:/smoke/NexusDesk',
        name: 'NexusDesk Smoke',
        truncated: false,
        scan: {
            included: 2,
            ignored: 0,
            depthSkipped: 0,
            entrySkipped: 0,
            unreadable: 0,
            maxDepth: 10,
            maxEntries: 800,
            ignoredSamples: [],
            skippedSamples: [],
        },
        nodes: [
            {name: 'campaigns.csv', path: 'E:/smoke/NexusDesk/data/campaigns.csv', relPath: 'data/campaigns.csv', kind: 'file', fileType: 'data', depth: 1, meta: 'csv / 128 B'},
            {name: 'brief.md', path: 'E:/smoke/NexusDesk/docs/brief.md', relPath: 'docs/brief.md', kind: 'file', fileType: 'document', depth: 1, meta: 'markdown / 80 B'},
        ],
    };
    const table = {
        columns: ['channel', 'spend', 'conversions'],
        rows: [['Search', '120', '8'], ['Email', '40', '6']],
        profiles: [
            {name: 'channel', type: 'text', missing: 0, distinct: 2},
            {name: 'spend', type: 'integer', missing: 0, distinct: 2, min: '40', max: '120'},
            {name: 'conversions', type: 'integer', missing: 0, distinct: 2, min: '6', max: '8'},
        ],
        totalRows: 2,
        truncated: false,
    };
    const metadataBrowser = {
        path: 'E:/smoke/NexusDesk/.nexusdesk/metadata/nexusdesk.sqlite',
        message: 'SQLite metadata tables and dataset SQL views are available for inspection.',
        updatedAt: new Date().toISOString(),
        tables: [
            {name: 'chats', rowCount: 2, columns: [{name: 'role', type: 'TEXT'}, {name: 'content', type: 'TEXT'}], sampleRows: [['assistant', 'Smoke answer']]},
            {name: 'artifacts', rowCount: 1, columns: [{name: 'rel_path', type: 'TEXT'}, {name: 'kind', type: 'TEXT'}], sampleRows: [['.nexusdesk/artifacts/smoke.md', 'chat-answer']]},
        ],
        datasetViews: [{name: 'campaigns', relPath: 'data/campaigns.csv', engine: 'duckdb view / csv fallback', columns: table.columns, rows: 2, message: 'campaigns view'}],
    };
    window.go = {
        main: {
            App: {
                GetStartupState: async () => ({
                    productName: 'NexusDesk',
                    tagline: 'Local-first AI IDE, data studio, and analytics studio.',
                    buildStage: 'Visual Smoke',
                    capabilities: [],
                    workspaceItems: [],
                    toolEvents: [],
                }),
                GetRecentWorkspaces: async () => [],
                GetLLMSettings: async () => ({providerName: 'Local', baseUrl: 'http://localhost:11434/v1', model: 'qwen3:8b', apiKey: '', updatedAt: ''}),
                SelectWorkspace: async () => ({selected: true, snapshot}),
                RefreshWorkspace: async () => ({selected: true, snapshot}),
                GetChatHistory: async () => [
                    {role: 'assistant', content: 'Smoke answer\n\nSources:\n- data/campaigns.csv', contextRelPath: 'data/campaigns.csv', sourcePaths: ['data/campaigns.csv'], createdAt: '2026-05-14T00:00:00Z'},
                ],
                ListArtifacts: async () => [{relPath: '.nexusdesk/artifacts/smoke.md', name: 'smoke.md', path: 'E:/smoke/NexusDesk/.nexusdesk/artifacts/smoke.md', kind: 'chat-answer', size: 120, modifiedAt: '2026-05-14T00:00:00Z', source: 'chat', summary: 'Smoke artifact', model: 'qwen3:8b'}],
                ListApprovals: async () => [],
                ListDatasetProfiles: async () => [{relPath: 'data/campaigns.csv', name: 'campaigns.csv', kind: 'csv', rows: 2, columns: 3, sheets: [], profiles: table.profiles, updatedAt: '2026-05-14T00:00:00Z', message: 'Profile ready'}],
                ListDatasetQueries: async () => [],
                ListDatasetSQLQueries: async () => [{relPath: 'data/campaigns.csv', query: 'select * from dataset', label: 'All campaigns', kind: 'sql', updatedAt: '2026-05-14T00:00:00Z'}],
                ListAgentToolRuns: async () => [{id: 'smoke-tool', toolName: 'dataset.query', title: 'Query Dataset', target: 'data/campaigns.csv', risk: 'low', requiresApproval: false, status: 'dry-run', mode: 'dry-run', inputs: {query: 'spend > 10'}, outputSummary: 'Ready to query dataset.', error: '', approvalId: 'approval-smoke', startedAt: '2026-05-14T00:00:00Z', completedAt: '2026-05-14T00:00:01Z', durationMs: 1000}],
                ListAgentTools: async () => [],
                ReadWorkspaceFile: async () => ({relPath: 'data/campaigns.csv', name: 'campaigns.csv', kind: 'file', fileType: 'data', content: 'channel,spend,conversions\nSearch,120,8\nEmail,40,6\n', text: 'channel,spend,conversions\nSearch,120,8\nEmail,40,6\n', encoding: 'utf-8', table, truncated: false, message: 'CSV preview ready', size: 64}),
                CheckWorkspaceFreshness: async () => ({changed: [{relPath: 'data/campaigns.csv', kind: 'modified', message: 'data/campaigns.csv changed on disk.'}], staleArtifacts: ['.nexusdesk/artifacts/smoke.md'], staleDatasets: ['data/campaigns.csv'], message: '1 workspace file changes detected. 1 artifacts may be stale. 1 dataset-derived views need refresh.'}),
                RefreshStaleContext: async () => ({preview: {roots: ['data/campaigns.csv'], files: [{relPath: 'data/campaigns.csv', required: true}], fileCount: 1, truncated: false, message: 'Context refreshed'}, affectedChats: 1, staleArtifacts: ['.nexusdesk/artifacts/smoke.md'], staleDatasets: ['data/campaigns.csv'], message: 'Refreshed context preview for 1 changed roots.'}),
                GetArtifactLineage: async () => ({message: '4 lineage nodes and 3 relationships.', relationshipCounts: {cited: 1, 'dry-run': 1, generated: 1}, nodes: [{id: 'source:data/campaigns.csv', kind: 'source', label: 'campaigns.csv', relPath: 'data/campaigns.csv'}, {id: 'chat:assistant:0', kind: 'chat', label: 'Assistant answer', relPath: 'data/campaigns.csv'}, {id: 'tool:smoke-tool', kind: 'tool', label: 'Query Dataset', relPath: 'data/campaigns.csv'}, {id: 'artifact:.nexusdesk/artifacts/smoke.md', kind: 'artifact', label: 'smoke.md', relPath: '.nexusdesk/artifacts/smoke.md'}], edges: [{from: 'source:data/campaigns.csv', to: 'chat:assistant:0', label: 'cited'}, {from: 'source:data/campaigns.csv', to: 'tool:smoke-tool', label: 'dry-run'}, {from: 'chat:assistant:0', to: 'artifact:.nexusdesk/artifacts/smoke.md', label: 'generated'}]}),
                InspectMetadataStore: async () => metadataBrowser,
                EnsureSQLiteMetadataStore: async () => ({path: metadataBrowser.path, schemaPath: 'schema.sql', schemaVersion: 1, schemaHash: 'smoke', tables: metadataBrowser.tables.map((table) => table.name), message: 'SQLite metadata store mirrored from current JSON compatibility stores.', updatedAt: metadataBrowser.updatedAt}),
                QueryDatasetSQL: async () => ({relPath: 'data/campaigns.csv', sql: 'select * from dataset', engine: 'duckdb-compatible-csv', columns: table.columns, rows: table.rows, totalRows: 2, matchedRows: 2, message: 'SQL smoke ready'}),
                SaveDatasetSQLQuery: async () => ({relPath: 'data/campaigns.csv', query: 'select * from dataset', label: 'All campaigns', kind: 'sql', updatedAt: '2026-05-14T00:00:00Z'}),
            },
        },
    };
});
await page.goto(server.url);
await page.getByText('Open Folder').click();
await page.locator('.metadata-store-panel').filter({hasText: 'Artifact Lineage'}).getByText('Refresh').click();
await page.getByText('Inspect metadata').click();
await page.locator('.tool-run-row summary').first().click();
await page.locator('.tool-run-detail').first().scrollIntoViewIfNeeded();
await page.locator('.tool-run-detail').getByText('Replay dry run').waitFor({state: 'visible'});
await page.getByLabel('Lineage filter').getByText('source').click();
const beforeResize = await page.locator('.navigator').boundingBox();
const resizer = await page.locator('.navigator-resizer').boundingBox();
if (!beforeResize || !resizer) {
    throw new Error('NexusDesk visual smoke failed: navigator or resizer is missing.');
}
await page.mouse.move(resizer.x + resizer.width / 2, resizer.y + 20);
await page.mouse.down();
await page.mouse.move(resizer.x + 80, resizer.y + 20);
await page.mouse.up();
const afterResize = await page.locator('.navigator').boundingBox();
if (!afterResize || Math.abs(afterResize.width - beforeResize.width) < 20) {
    throw new Error('NexusDesk visual smoke failed: navigator resizing did not change width.');
}
const hasBodyScroll = await page.evaluate(() => document.scrollingElement ? document.scrollingElement.scrollHeight > window.innerHeight + 2 : false);
if (hasBodyScroll) {
    throw new Error('NexusDesk visual smoke failed: whole window became scrollable.');
}
for (const text of ['Metadata Browser', 'Workspace Watcher', 'Artifact Lineage', 'Replay dry run', 'Context changed since this answer was created.', 'All campaigns']) {
    if (!(await page.getByText(text).first().isVisible())) {
        throw new Error(`NexusDesk visual smoke failed: missing ${text}.`);
    }
}
await page.screenshot({path: path.join(screenshotDir, 'desktop.png'), fullPage: false});
await page.screenshot({path: path.join(baselineDir, 'desktop.png'), fullPage: false});
await page.setViewportSize({width: 390, height: 844});
await page.screenshot({path: path.join(screenshotDir, 'mobile.png'), fullPage: false});
await page.screenshot({path: path.join(baselineDir, 'mobile.png'), fullPage: false});
await browser.close();
await server.close();

writeFileSync(path.join(baselineDir, 'manifest.json'), `${JSON.stringify({
    generatedAt: new Date().toISOString(),
    source: 'dist/index.html',
    viewports: ['desktop', 'mobile'],
    assertions: ['navigator-resize', 'panel-overflow', 'tool-run-detail', 'lineage-filter', 'lineage-graph', 'freshness-warning', 'metadata-browser', 'sql-snippets'],
}, null, 2)}\n`);

console.log('NexusDesk visual smoke captured desktop/mobile screenshots and baseline metadata.');

async function serveDist(distRoot) {
    const server = http.createServer((request, response) => {
        const requestPath = decodeURIComponent(new URL(request.url ?? '/', 'http://127.0.0.1').pathname);
        const safePath = path.normalize(requestPath).replace(/^(\.\.[/\\])+/, '');
        let filePath = path.join(distRoot, safePath === '/' ? 'index.html' : safePath);
        if (!filePath.startsWith(distRoot)) {
            response.writeHead(403);
            response.end('Forbidden');
            return;
        }
        try {
            if (statSync(filePath).isDirectory()) {
                filePath = path.join(filePath, 'index.html');
            }
            response.writeHead(200, {'content-type': contentType(filePath)});
            response.end(readFileSync(filePath));
        } catch {
            response.writeHead(404);
            response.end('Not found');
        }
    });
    await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
    const address = server.address();
    return {
        url: `http://127.0.0.1:${address.port}/`,
        close: () => new Promise((resolve) => server.close(resolve)),
    };
}

function contentType(filePath) {
    if (filePath.endsWith('.html')) {
        return 'text/html';
    }
    if (filePath.endsWith('.js')) {
        return 'text/javascript';
    }
    if (filePath.endsWith('.css')) {
        return 'text/css';
    }
    if (filePath.endsWith('.svg')) {
        return 'image/svg+xml';
    }
    if (filePath.endsWith('.ttf')) {
        return 'font/ttf';
    }
    return 'application/octet-stream';
}
