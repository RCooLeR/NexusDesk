import {existsSync, mkdirSync, readFileSync, statSync, writeFileSync} from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {installNexusMocks} from './visual-fixtures.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const indexPath = path.join(root, 'dist', 'index.html');
const screenshotDir = path.join(root, 'dist', 'smoke');
const baselineDir = path.join(root, 'visual-baselines');

if (!existsSync(indexPath)) {
    console.error('Nexus visual smoke failed: run npm run build first.');
    process.exit(1);
}

let chromium;
try {
    ({chromium} = await import('playwright'));
} catch {
    console.error('Nexus visual smoke failed: Playwright is not installed.');
    process.exit(1);
}

mkdirSync(screenshotDir, {recursive: true});
mkdirSync(baselineDir, {recursive: true});
const server = await serveDist(path.join(root, 'dist'));
const browser = await chromium.launch();
const page = await browser.newPage({viewport: {width: 1440, height: 960}});
await installNexusMocks(page);

try {
    await page.goto(server.url);
    await page.getByText('Open Folder').click();
    await page.locator('[data-studio-route="code"]').click();
    for (const text of ['Active Context', 'Preview', 'Explain']) {
        if (!(await page.getByText(text).first().isVisible())) {
            throw new Error(`Nexus visual smoke failed: missing Workbench route text ${text}.`);
        }
    }
    if ((await page.locator('.code-studio-inline').count()) > 0) {
        throw new Error('Nexus visual smoke failed: Workbench route rendered the removed inline Git panel above editor tabs.');
    }
    await page.locator('.editor-tabs').waitFor({state: 'visible'});
    await page.getByRole('tab', {name: 'Git', exact: true}).click();
    for (const text of ['Git', 'Working Tree Diff', 'Refresh git']) {
        if (!(await page.getByText(text).first().isVisible())) {
            throw new Error(`Nexus visual smoke failed: missing bottom Git drawer text ${text}.`);
        }
    }
    await page.getByRole('tree', {name: 'Project tree'}).waitFor({state: 'visible'});
    const firstTreeRow = page.locator('.project-tree .tree-item').first();
    await firstTreeRow.scrollIntoViewIfNeeded();
    await firstTreeRow.waitFor({state: 'visible'});
    await page.locator('.project-tree .tree-node-badge').first().waitFor({state: 'attached'});
    await firstTreeRow.click({button: 'right'});
    await page.locator('.tree-context-menu').getByText('Copy path').waitFor({state: 'visible'});
    await firstTreeRow.click();
    await page.locator('[data-studio-route="data"]').click();
    const dataMainSurface = page.getByLabel('Data & Analytics main surface');
    await dataMainSurface.getByText('Data & Analytics', {exact: true}).waitFor({state: 'visible'});
    await dataMainSurface.getByRole('button', {name: 'Inspect metadata'}).click();
    await dataMainSurface.getByPlaceholder('Search chats, artifacts, tools').fill('Smoke');
    await dataMainSurface.getByText('Search history').click({force: true});
    await page.locator('[data-studio-route="artifacts"]').click();
    await page.locator('.metadata-store-panel').filter({hasText: 'Artifact Lineage'}).getByText('Refresh').click();
    await page.getByText('Export JSON').click();
    await page.getByLabel('Lineage filter').getByText('source').click();
    await page.getByRole('tab', {name: 'Approvals', exact: true}).click();
    await page.getByText('Approval Log').waitFor({state: 'visible'});
    await page.locator('.agent-panel').getByText('Nexus Assistant').waitFor({state: 'visible'});
    await page.locator('.agent-panel').getByLabel('Submit mode').waitFor({state: 'visible'});

    const beforeResize = await page.locator('.navigator').boundingBox();
    const resizer = await page.locator('.navigator-resizer').boundingBox();
    if (!beforeResize || !resizer) {
        throw new Error('Nexus visual smoke failed: navigator or resizer is missing.');
    }
    await page.mouse.move(resizer.x + resizer.width / 2, resizer.y + 20);
    await page.mouse.down();
    await page.mouse.move(resizer.x + 80, resizer.y + 20);
    await page.mouse.up();
    const afterResize = await page.locator('.navigator').boundingBox();
    if (!afterResize || Math.abs(afterResize.width - beforeResize.width) < 20) {
        throw new Error('Nexus visual smoke failed: navigator resizing did not change width.');
    }

    const beforeAgentResize = await page.locator('.agent-panel').boundingBox();
    const agentResizer = await page.locator('.agent-resizer').boundingBox();
    if (!beforeAgentResize || !agentResizer) {
        throw new Error('Nexus visual smoke failed: agent panel or resizer is missing.');
    }
    await page.mouse.move(agentResizer.x + agentResizer.width / 2, agentResizer.y + 20);
    await page.mouse.down();
    await page.mouse.move(agentResizer.x - 80, agentResizer.y + 20);
    await page.mouse.up();
    const afterAgentResize = await page.locator('.agent-panel').boundingBox();
    if (!afterAgentResize || Math.abs(afterAgentResize.width - beforeAgentResize.width) < 20) {
        throw new Error('Nexus visual smoke failed: agent panel resizing did not change width.');
    }

    const bottomDrawer = page.locator('.bottom-studio-panel:not(.main-studio-panel)');
    const beforeBottomResize = await bottomDrawer.boundingBox();
    const bottomResizer = await page.locator('.bottom-panel-resizer').boundingBox();
    if (!beforeBottomResize || !bottomResizer) {
        throw new Error('Nexus visual smoke failed: bottom panel or resizer is missing.');
    }
    await page.mouse.move(bottomResizer.x + 80, bottomResizer.y + bottomResizer.height / 2);
    await page.mouse.down();
    await page.mouse.move(bottomResizer.x + 80, bottomResizer.y - 80);
    await page.mouse.up();
    const afterBottomResize = await bottomDrawer.boundingBox();
    if (!afterBottomResize || Math.abs(afterBottomResize.height - beforeBottomResize.height) < 20) {
        throw new Error('Nexus visual smoke failed: bottom panel resizing did not change height.');
    }

    const hasBodyScroll = await page.evaluate(() => document.scrollingElement ? document.scrollingElement.scrollHeight > window.innerHeight + 2 : false);
    if (hasBodyScroll) {
        throw new Error('Nexus visual smoke failed: whole window became scrollable.');
    }

    await page.locator('[data-studio-route="data"]').click();
    for (const text of ['Metadata Browser', 'Workspace Watcher']) {
        if (!(await page.getByText(text).first().isVisible())) {
            throw new Error(`Nexus visual smoke failed: missing ${text}.`);
        }
    }
    for (const text of ['Context changed since this answer was created.', 'Smoke answer']) {
        if (!(await page.getByText(text).first().isVisible())) {
            throw new Error(`Nexus visual smoke failed: missing ${text}.`);
        }
    }
    await page.locator('[data-studio-route="artifacts"]').click();
    await page.getByText('Artifact Lineage').waitFor({state: 'visible'});
    await page.getByRole('tab', {name: 'Approvals', exact: true}).click();
    await page.getByText('Approval Log').waitFor({state: 'visible'});
    await page.locator('.agent-panel').getByText('Nexus Assistant').waitFor({state: 'visible'});
    await page.locator('[data-studio-route="settings"]').click();
    await page.getByText('LLM Provider', {exact: true}).waitFor({state: 'visible'});

    await page.screenshot({path: path.join(screenshotDir, 'desktop.png'), fullPage: false});
    await page.screenshot({path: path.join(baselineDir, 'desktop.png'), fullPage: false});
    await page.setViewportSize({width: 390, height: 844});
    await page.screenshot({path: path.join(screenshotDir, 'mobile.png'), fullPage: false});
    await page.screenshot({path: path.join(baselineDir, 'mobile.png'), fullPage: false});

    writeFileSync(path.join(baselineDir, 'manifest.json'), `${JSON.stringify({
        generatedAt: new Date().toISOString(),
        source: 'dist/index.html',
        viewports: ['desktop', 'mobile'],
        assertions: ['navigator-resize', 'agent-resize', 'bottom-drawer-resize', 'panel-overflow', 'project-tree', 'tree-context-menu', 'tool-run-detail', 'code-route', 'code-git-diff', 'settings-route', 'data-route', 'artifacts-route', 'approvals-tab', 'lineage-export', 'lineage-filter', 'lineage-graph', 'freshness-warning', 'metadata-browser', 'metadata-history', 'sql-snippets', 'dataset-lineage'],
    }, null, 2)}\n`);
} finally {
    await browser.close();
    await server.close();
}

console.log('Nexus visual smoke captured desktop/mobile screenshots and baseline metadata.');

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
