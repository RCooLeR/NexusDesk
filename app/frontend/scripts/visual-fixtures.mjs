export async function installNexusDeskMocks(page) {
    await page.addInitScript(() => {
        const snapshot = {
            root: 'E:/smoke/NexusDesk',
            name: 'NexusDesk Smoke',
            truncated: false,
            scan: {
                included: 3,
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
                {name: 'local.sqlite', path: 'E:/smoke/NexusDesk/data/local.sqlite', relPath: 'data/local.sqlite', kind: 'file', fileType: 'database', depth: 1, meta: 'sqlite / 4 KiB'},
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
        const sqlRuns = [
            {id: 'sql-smoke', relPath: 'data/campaigns.csv', sql: 'select * from dataset', engine: 'duckdb-compatible-csv', status: 'ok', message: 'SQL smoke ready', rowCount: 2, artifactRelPath: '.nexusdesk/artifacts/sql-smoke.md', createdAt: '2026-05-14T00:00:00Z'},
        ];
        const dependencies = [
            {id: 'dep-smoke', relPath: 'data/campaigns.csv', kind: 'sql-report', query: 'select * from dataset', target: '.nexusdesk/artifacts/sql-smoke.md', artifactRelPath: '.nexusdesk/artifacts/sql-smoke.md', createdAt: '2026-05-14T00:00:00Z'},
        ];
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
                    ListDatasetDependencies: async () => dependencies,
                    ListDatasetSQLRuns: async () => sqlRuns,
                    ListAgentToolRuns: async () => [{id: 'smoke-tool', toolName: 'dataset.query', title: 'Query Dataset', target: 'data/campaigns.csv', risk: 'low', requiresApproval: false, status: 'dry-run', mode: 'dry-run', inputs: {query: 'spend > 10'}, outputSummary: 'Ready to query dataset.', error: '', approvalId: 'approval-smoke', startedAt: '2026-05-14T00:00:00Z', completedAt: '2026-05-14T00:00:01Z', durationMs: 1000}],
                    ListAgentTools: async () => [],
                    ReadWorkspaceFile: async (relPath) => {
                        if (relPath === 'data/local.sqlite') {
                            return {relPath, name: 'local.sqlite', kind: 'file', fileType: 'database', content: '', text: '', encoding: '', truncated: false, message: 'SQLite database file ready for read-only connector queries.', size: 4096};
                        }
                        return {relPath: 'data/campaigns.csv', name: 'campaigns.csv', kind: 'file', fileType: 'data', content: 'channel,spend,conversions\nSearch,120,8\nEmail,40,6\n', text: 'channel,spend,conversions\nSearch,120,8\nEmail,40,6\n', encoding: 'utf-8', table, truncated: false, message: 'CSV preview ready', size: 64};
                    },
                    CheckWorkspaceFreshness: async () => ({changed: [{relPath: 'data/campaigns.csv', kind: 'modified', message: 'data/campaigns.csv changed on disk.'}], staleArtifacts: ['.nexusdesk/artifacts/smoke.md'], staleDatasets: ['data/campaigns.csv'], message: '1 workspace file changes detected. 1 artifacts may be stale. 1 dataset-derived views need refresh.'}),
                    RefreshStaleContext: async () => ({preview: {roots: ['data/campaigns.csv'], files: [{relPath: 'data/campaigns.csv', required: true}], fileCount: 1, truncated: false, message: 'Context refreshed'}, affectedChats: 1, staleArtifacts: ['.nexusdesk/artifacts/smoke.md'], staleDatasets: ['data/campaigns.csv'], message: 'Refreshed context preview for 1 changed roots.'}),
                    GetArtifactLineage: async () => ({message: '4 lineage nodes and 3 relationships.', relationshipCounts: {cited: 1, 'dry-run': 1, generated: 1}, nodes: [{id: 'source:data/campaigns.csv', kind: 'source', label: 'campaigns.csv', relPath: 'data/campaigns.csv'}, {id: 'chat:assistant:0', kind: 'chat', label: 'Assistant answer', relPath: 'data/campaigns.csv'}, {id: 'tool:smoke-tool', kind: 'tool', label: 'Query Dataset', relPath: 'data/campaigns.csv'}, {id: 'artifact:.nexusdesk/artifacts/smoke.md', kind: 'artifact', label: 'smoke.md', relPath: '.nexusdesk/artifacts/smoke.md'}], edges: [{from: 'source:data/campaigns.csv', to: 'chat:assistant:0', label: 'cited'}, {from: 'source:data/campaigns.csv', to: 'tool:smoke-tool', label: 'dry-run'}, {from: 'chat:assistant:0', to: 'artifact:.nexusdesk/artifacts/smoke.md', label: 'generated'}]}),
                    ExportArtifactLineageJSON: async () => ({relPath: '.nexusdesk/artifacts/lineage-smoke.json', name: 'lineage-smoke.json', kind: 'lineage-json', size: 512, path: 'E:/smoke/NexusDesk/.nexusdesk/artifacts/lineage-smoke.json'}),
                    ImportArtifactLineageJSON: async () => ({lineage: {nodes: [], edges: []}, message: 'Imported lineage preview.'}),
                    InspectMetadataStore: async () => metadataBrowser,
                    EnsureSQLiteMetadataStore: async () => ({path: metadataBrowser.path, schemaPath: 'schema.sql', schemaVersion: 1, schemaHash: 'smoke', tables: metadataBrowser.tables.map((table) => table.name), message: 'SQLite metadata store mirrored from current JSON compatibility stores.', updatedAt: metadataBrowser.updatedAt}),
                    SearchMetadata: async () => [{kind: 'chat', relPath: 'data/campaigns.csv', title: 'Assistant answer', snippet: 'Smoke answer', createdAt: '2026-05-14T00:00:00Z'}],
                    QueryDatasetSQL: async () => ({relPath: 'data/campaigns.csv', sql: 'select * from dataset', engine: 'duckdb-compatible-csv', columns: table.columns, rows: table.rows, totalRows: 2, matchedRows: 2, message: 'SQL smoke ready'}),
                    SaveDatasetSQLQuery: async () => ({relPath: 'data/campaigns.csv', query: 'select * from dataset', label: 'All campaigns', kind: 'sql', updatedAt: '2026-05-14T00:00:00Z'}),
                    QueryWorkspaceSQLite: async () => ({relPath: 'data/local.sqlite', sql: 'select name, type from sqlite_master', columns: ['name', 'type'], rows: [['campaigns', 'table']], totalRows: 1, truncated: false, message: 'Read-only SQLite query returned 1 row.'}),
                },
            },
        };
    });
}
