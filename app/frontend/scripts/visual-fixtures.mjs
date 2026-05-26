export async function installNexusMocks(page) {
    await page.addInitScript(() => {
        const snapshot = {
            root: 'E:/smoke/NexusAugenticStudio',
            name: 'Nexus Smoke',
            truncated: false,
            scan: {
                included: 5,
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
                {name: 'data', path: 'E:/smoke/NexusAugenticStudio/data', relPath: 'data', kind: 'directory', fileType: 'folder', depth: 1, meta: '2 files'},
                {name: 'campaigns.csv', path: 'E:/smoke/NexusAugenticStudio/data/campaigns.csv', relPath: 'data/campaigns.csv', kind: 'file', fileType: 'data', depth: 2, meta: 'csv / 128 B'},
                {name: 'local.sqlite', path: 'E:/smoke/NexusAugenticStudio/data/local.sqlite', relPath: 'data/local.sqlite', kind: 'file', fileType: 'database', depth: 2, meta: 'sqlite / 4 KiB'},
                {name: 'docs', path: 'E:/smoke/NexusAugenticStudio/docs', relPath: 'docs', kind: 'directory', fileType: 'folder', depth: 1, meta: '1 file'},
                {name: 'brief.md', path: 'E:/smoke/NexusAugenticStudio/docs/brief.md', relPath: 'docs/brief.md', kind: 'file', fileType: 'document', depth: 2, meta: 'markdown / 80 B'},
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
            path: 'E:/smoke/NexusAugenticStudio/.nexusdesk/metadata/nexus.sqlite',
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
                        productName: 'Nexus Augentic Studio',
                        tagline: 'Agentic work. Augmented by context.',
                        buildStage: 'Visual Smoke',
                        capabilities: [],
                        workspaceItems: [],
                        toolEvents: [],
                    }),
                    GetRecentWorkspaces: async () => [],
                    GetLLMSettings: async () => ({providerName: 'Local', baseUrl: 'http://localhost:11434/v1', model: 'qwen3:8b', apiKey: '', maxContextTokens: 32768, responseReserveTokens: 4096, updatedAt: ''}),
                    ListConnectorProfiles: async () => [{id: 'postgres-smoke', name: 'Smoke warehouse', kind: 'postgres', driver: 'postgres', host: 'db.smoke.local', port: 5432, database: 'analytics', username: 'analyst', password: '********', credentialRef: 'nexus:connector-profile:postgres-smoke:password', sslMode: 'prefer', workspaceScope: '', readOnly: true, resultLimit: 1000, timeoutSeconds: 30, updatedAt: '2026-05-14T00:00:00Z'}],
                    SaveConnectorProfile: async (profile) => ({...profile, id: profile.id || 'postgres-smoke', password: profile.password ? '********' : '', credentialRef: profile.password ? 'nexus:connector-profile:postgres-smoke:password' : '', readOnly: true, updatedAt: '2026-05-14T00:00:00Z'}),
                    DeleteConnectorProfile: async () => undefined,
                    SelectWorkspace: async () => ({selected: true, snapshot}),
                    RefreshWorkspace: async () => ({selected: true, snapshot}),
                    GetGitStatus: async () => {
                        throw new Error('Git status should only load after an explicit Refresh git action.');
                    },
                    GetGitFileDiff: async (relPath) => ({
                        path: relPath,
                        stagedDiff: '',
                        stagedDiffTruncated: false,
                        unstagedDiff: `diff --git a/${relPath} b/${relPath}\n@@\n- old\n+ new\n`,
                        unstagedDiffTruncated: false,
                        message: `Loaded read-only diff for ${relPath}.`,
                        generatedAt: '2026-05-14T00:00:00Z',
                    }),
                    GetChatHistory: async () => [
                        {role: 'assistant', content: 'Smoke answer\n\nSources:\n- data/campaigns.csv', contextRelPath: 'data/campaigns.csv', sourcePaths: ['data/campaigns.csv'], createdAt: '2026-05-14T00:00:00Z'},
                    ],
                    ListArtifacts: async () => [{relPath: '.nexusdesk/artifacts/smoke.md', name: 'smoke.md', path: 'E:/smoke/NexusAugenticStudio/.nexusdesk/artifacts/smoke.md', kind: 'chat-answer', size: 120, modifiedAt: '2026-05-14T00:00:00Z', source: 'chat', summary: 'Smoke artifact', model: 'qwen3:8b'}],
                    ListApprovals: async () => [],
                    ListDatasetProfiles: async () => [{relPath: 'data/campaigns.csv', name: 'campaigns.csv', kind: 'csv', rows: 2, columns: 3, sheets: [], profiles: table.profiles, updatedAt: '2026-05-14T00:00:00Z', message: 'Profile ready'}],
                    ListDatasetQueries: async () => [],
                    ListDatasetSQLQueries: async () => [{relPath: 'data/campaigns.csv', query: 'select * from dataset', label: 'All campaigns', kind: 'sql', updatedAt: '2026-05-14T00:00:00Z'}],
                    ListDatasetDependencies: async () => dependencies,
                    ListDatasetSQLRuns: async () => sqlRuns,
                    ListWorkspaceTasks: async () => ({
                        tasks: [
                            {id: 'npm-app-frontend-build', kind: 'npm-script', label: 'npm run build', command: 'npm run build', cwd: 'app/frontend', source: 'app/frontend/package.json'},
                            {id: 'go-app-test-all', kind: 'go-test', label: 'go test ./...', command: 'go test ./...', cwd: 'app', source: 'app/go.mod'},
                        ],
                        message: '2 tasks detected from package scripts and Go tests.',
                        generatedAt: '2026-05-14T00:00:00Z',
                    }),
                    RunWorkspaceTask: async (request) => ({
                        task: {id: request.taskId, kind: 'go-test', label: 'go test ./...', command: 'go test ./...', cwd: 'app', source: 'app/go.mod'},
                        status: 'success',
                        exitCode: 0,
                        stdout: 'ok task smoke',
                        stderr: '',
                        startedAt: '2026-05-14T00:00:00Z',
                        completedAt: '2026-05-14T00:00:01Z',
                        durationMs: 1000,
                        artifactRelPath: '.nexusdesk/artifacts/task-run-smoke.md',
                        message: 'Task run artifact created.',
                    }),
                    ListWorkspaceProblems: async () => ({
                        problems: [
                            {relPath: 'docs/brief.md', name: 'brief.md', severity: 'info', source: 'marker', message: 'Task marker: TODO: tighten this brief', line: 3},
                        ],
                        message: '1 workspace problem detected by lightweight scanners.',
                        generatedAt: '2026-05-14T00:00:00Z',
                        truncated: false,
                    }),
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
                    ExportArtifactLineageJSON: async () => ({relPath: '.nexusdesk/artifacts/lineage-smoke.json', name: 'lineage-smoke.json', kind: 'lineage-json', size: 512, path: 'E:/smoke/NexusAugenticStudio/.nexusdesk/artifacts/lineage-smoke.json'}),
                    ImportArtifactLineageJSON: async () => ({lineage: {nodes: [], edges: []}, message: 'Imported lineage preview.'}),
                    InspectMetadataStore: async () => metadataBrowser,
                    EnsureSQLiteMetadataStore: async () => ({path: metadataBrowser.path, schemaPath: 'schema.sql', schemaVersion: 1, schemaHash: 'smoke', tables: metadataBrowser.tables.map((table) => table.name), message: 'SQLite metadata store mirrored from current JSON compatibility stores.', updatedAt: metadataBrowser.updatedAt}),
                    SearchMetadata: async () => [{kind: 'chat', relPath: 'data/campaigns.csv', title: 'Assistant answer', snippet: 'Smoke answer', createdAt: '2026-05-14T00:00:00Z'}],
                    QueryDatasetSQL: async () => ({relPath: 'data/campaigns.csv', sql: 'select * from dataset', engine: 'duckdb-compatible-csv', columns: table.columns, rows: table.rows, totalRows: 2, matchedRows: 2, message: 'SQL smoke ready'}),
                    SaveDatasetSQLQuery: async () => ({relPath: 'data/campaigns.csv', query: 'select * from dataset', label: 'All campaigns', kind: 'sql', updatedAt: '2026-05-14T00:00:00Z'}),
                    QueryWorkspaceSQLite: async () => ({relPath: 'data/local.sqlite', sql: 'select name, type from sqlite_master', columns: ['name', 'type'], rows: [['campaigns', 'table']], totalRows: 1, truncated: false, message: 'Read-only SQLite query returned 1 row.'}),
                    InspectWorkspaceSQLite: async () => ({
                        id: 'sqlite:data/local.sqlite',
                        relPath: 'data/local.sqlite',
                        name: 'local.sqlite',
                        kind: 'sqlite',
                        engine: 'sqlite-readonly',
                        readOnly: true,
                        tables: [{name: 'campaigns', type: 'table', rowCount: 2, columns: [{name: 'channel', type: 'TEXT', nullable: false, primaryKey: false, default: ''}, {name: 'spend', type: 'INTEGER', nullable: true, primaryKey: false, default: ''}], indexes: [{name: 'campaigns_channel_idx', table: 'campaigns', unique: false, columns: ['channel']}], sampleRows: [['Search', '120'], ['Email', '40']]}],
                        views: [],
                        indexes: [{name: 'campaigns_channel_idx', table: 'campaigns', unique: false, columns: ['channel']}],
                        message: 'SQLite connector metadata inspected: 1 tables, 0 views.',
                    }),
                },
            },
        };
    });
}
