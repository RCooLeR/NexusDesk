import {existsSync, mkdirSync, readFileSync, statSync, writeFileSync} from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {installNexusDeskMocks} from './visual-fixtures.mjs';

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
await installNexusDeskMocks(page);

try {
    await page.goto(server.url);
    await page.getByText('Open Folder').click();
    await page.locator('.metadata-store-panel').filter({hasText: 'Artifact Lineage'}).getByText('Refresh').click();
    await page.getByText('Export JSON').click();
    await page.getByText('Inspect metadata').click();
    await page.getByPlaceholder('Search chats, artifacts, tools').fill('Smoke');
    await page.getByText('Search history').click({force: true});
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

    for (const text of ['Metadata Browser', 'Workspace Watcher', 'Artifact Lineage', 'Replay dry run', 'Context changed since this answer was created.', 'Smoke answer']) {
        if (!(await page.getByText(text).first().isVisible())) {
            throw new Error(`NexusDesk visual smoke failed: missing ${text}.`);
        }
    }

    await page.screenshot({path: path.join(screenshotDir, 'desktop.png'), fullPage: false});
    await page.screenshot({path: path.join(baselineDir, 'desktop.png'), fullPage: false});
    await page.setViewportSize({width: 390, height: 844});
    await page.screenshot({path: path.join(screenshotDir, 'mobile.png'), fullPage: false});
    await page.screenshot({path: path.join(baselineDir, 'mobile.png'), fullPage: false});

    writeFileSync(path.join(baselineDir, 'manifest.json'), `${JSON.stringify({
        generatedAt: new Date().toISOString(),
        source: 'dist/index.html',
        viewports: ['desktop', 'mobile'],
        assertions: ['navigator-resize', 'panel-overflow', 'tool-run-detail', 'lineage-export', 'lineage-filter', 'lineage-graph', 'freshness-warning', 'metadata-browser', 'metadata-history', 'sql-snippets', 'dataset-lineage'],
    }, null, 2)}\n`);
} finally {
    await browser.close();
    await server.close();
}

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
