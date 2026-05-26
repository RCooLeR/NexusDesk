import {existsSync, mkdirSync, readFileSync, statSync, writeFileSync} from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';
import {installNexusMocks} from './visual-fixtures.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = path.resolve(root, '..', '..');
const indexPath = path.join(root, 'dist', 'index.html');
const screenshotDir = path.join(repoRoot, '.codex-ui-screenshots', 'loaded');

if (!existsSync(indexPath)) {
    console.error('Nexus visual gallery failed: run npm run build first.');
    process.exit(1);
}

let chromium;
try {
    ({chromium} = await import('playwright'));
} catch {
    console.error('Nexus visual gallery failed: Playwright is not installed.');
    process.exit(1);
}

mkdirSync(screenshotDir, {recursive: true});
const server = await serveDist(path.join(root, 'dist'));
const browser = await chromium.launch();
const page = await browser.newPage({viewport: {width: 1440, height: 960}});
await page.addInitScript(() => {
    window.localStorage.clear();
});
await installNexusMocks(page);

const captures = [];

try {
    await page.goto(server.url);
    await clickRole(page, 'button', 'Open Folder');
    await page.locator('.editor-tabs').waitFor({state: 'visible'});
    await capture(page, '01-workbench-compact-rail', 'Workbench / compact rail');

    await clickRole(page, 'button', 'Expand main navigation');
    await page.locator('.workspace-rail.expanded .rail-logo img').waitFor({state: 'visible'});
    await capture(page, '02-workbench-expanded-rail', 'Workbench / expanded rail');

    await openTreeContextMenu(page);
    await capture(page, '03-tree-context-menu', 'Project tree context menu');
    await page.keyboard.press('Escape');
    await page.locator('.tree-context-menu').waitFor({state: 'hidden'});

    await clickRole(page, 'button', 'Delete');
    await page.getByText('Approval Required').waitFor({state: 'visible'});
    await capture(page, '04-approval-modal', 'Approval modal');
    await clickRole(page, 'button', 'Cancel');

    await clickRole(page, 'tab', 'Git');
    await capture(page, '05-bottom-git', 'Bottom drawer / Git');
    await clickRole(page, 'tab', 'Approvals');
    await capture(page, '06-bottom-approvals', 'Bottom drawer / Approvals');
    await clickRole(page, 'tab', 'Activity');
    await capture(page, '07-bottom-activity', 'Bottom drawer / Activity');
    await page.getByLabel('Submit mode').selectOption('agent');
    await capture(page, '08-agent-write-toggle', 'Agent composer / write access toggle');

    await page.locator('[data-studio-route="data"]').click();
    await page.getByLabel('Data & Analytics main surface').getByText('Data & Analytics', {exact: true}).waitFor({state: 'visible'});
    await capture(page, '09-data-sources', 'Data & Analytics / Sources');
    await clickRouteTab(page, 'Data & Analytics main surface', 'Operations');
    await page.getByText('Workspace Watcher').waitFor({state: 'visible'});
    await capture(page, '10-data-operations', 'Data & Analytics / Operations');
    await clickRouteTab(page, 'Data & Analytics main surface', 'Metadata');
    await clickRole(page, 'button', 'Inspect metadata');
    await page.getByText('Metadata Browser').waitFor({state: 'visible'});
    await capture(page, '11-data-metadata', 'Data & Analytics / Metadata');

    await page.locator('[data-studio-route="artifacts"]').click();
    await page.getByLabel('Artifacts main surface').getByText('Artifacts', {exact: true}).waitFor({state: 'visible'});
    await capture(page, '12-artifacts-library', 'Artifacts / Library');
    await clickRouteTab(page, 'Artifacts main surface', 'Metadata');
    await capture(page, '13-artifacts-metadata', 'Artifacts / Metadata');
    await clickRouteTab(page, 'Artifacts main surface', 'Lineage');
    await page.getByLabel('Artifacts main surface').getByRole('button', {name: 'Refresh', exact: true}).click();
    await page.getByText('Artifact Lineage').waitFor({state: 'visible'});
    await capture(page, '14-artifacts-lineage', 'Artifacts / Lineage');

    await page.locator('[data-studio-route="settings"]').click();
    await page.getByText('LLM Provider', {exact: true}).waitFor({state: 'visible'});
    await capture(page, '15-settings-provider', 'Settings / Provider');
    await clickRouteTab(page, 'Settings main surface', 'Connectors');
    await clickRole(page, 'button', 'Inspect');
    await page.getByLabel('Settings main surface').locator('.connector-profile-metadata').waitFor({state: 'visible'});
    await capture(page, '16-settings-connectors', 'Settings / Connectors');
    await clickRouteTab(page, 'Settings main surface', 'Access');
    await page.getByText('Access & Approvals').waitFor({state: 'visible'});
    await capture(page, '17-settings-access', 'Settings / Access & Approvals');

    await page.keyboard.press('ControlOrMeta+Shift+P');
    await page.getByPlaceholder('Run command...').fill('git');
    await capture(page, '18-command-palette', 'Command palette');
    await page.keyboard.press('Escape');

    await page.keyboard.press('ControlOrMeta+P');
    await page.getByPlaceholder('Open file, folder, dataset, artifact...').fill('brief');
    await capture(page, '19-quick-open', 'Quick open');
    await page.keyboard.press('Escape');

    await page.setViewportSize({width: 390, height: 844});
    await page.locator('[data-studio-route="code"]').click();
    await page.locator('.editor-tabs').waitFor({state: 'visible'});
    await capture(page, '20-mobile-workbench', 'Mobile / Workbench');
    await page.locator('[data-studio-route="data"]').click();
    await capture(page, '21-mobile-data', 'Mobile / Data & Analytics');
    await page.locator('[data-studio-route="settings"]').click();
    await capture(page, '22-mobile-settings', 'Mobile / Settings');

    writeManifest();
    await writeContactSheet(browser);
} finally {
    await browser.close();
    await server.close();
}

console.log(`Nexus visual gallery captured ${captures.length} screenshots in ${screenshotDir}.`);

async function capture(targetPage, fileStem, title) {
    const fileName = `${fileStem}.png`;
    await targetPage.screenshot({path: path.join(screenshotDir, fileName), fullPage: false});
    captures.push({file: fileName, title});
}

async function clickRole(targetPage, role, name) {
    const locator = targetPage.getByRole(role, {name, exact: true});
    await locator.waitFor({state: 'visible'});
    await locator.click();
}

async function clickRouteTab(targetPage, surfaceLabel, tabName) {
    const surface = targetPage.getByLabel(surfaceLabel);
    await surface.getByRole('tab', {name: tabName, exact: true}).click();
}

async function openTreeContextMenu(targetPage) {
    const firstTreeRow = targetPage.locator('.project-tree .tree-item').first();
    await firstTreeRow.scrollIntoViewIfNeeded();
    await firstTreeRow.click({button: 'right'});
    await targetPage.locator('.tree-context-menu').waitFor({state: 'visible'});
}

function writeManifest() {
    writeFileSync(path.join(screenshotDir, 'manifest.json'), `${JSON.stringify({
        generatedAt: new Date().toISOString(),
        source: 'dist/index.html with visual-fixtures mocks',
        screenshots: captures,
    }, null, 2)}\n`);
}

async function writeContactSheet(activeBrowser) {
    const htmlPath = path.join(screenshotDir, 'index.html');
    writeFileSync(htmlPath, galleryHTML(captures));
    const sheetPage = await activeBrowser.newPage({viewport: {width: 1440, height: 1200}});
    await sheetPage.goto(pathToFileURL(htmlPath).href);
    await sheetPage.screenshot({path: path.join(screenshotDir, '00-contact-sheet.png'), fullPage: true});
    await sheetPage.close();
}

function galleryHTML(items) {
    return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8" />
<title>Nexus UI Visual Gallery</title>
<style>
body { margin: 0; font-family: Inter, Segoe UI, Arial, sans-serif; background: #f3f6fb; color: #111827; }
header { padding: 24px 28px 8px; }
h1 { font-size: 24px; margin: 0 0 6px; }
p { margin: 0; color: #475467; }
.grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; padding: 20px 28px 28px; }
figure { margin: 0; background: white; border: 1px solid #d9e2ee; border-radius: 8px; overflow: hidden; box-shadow: 0 8px 20px rgba(15, 23, 42, 0.08); }
figcaption { padding: 10px 12px; font-weight: 700; font-size: 13px; border-bottom: 1px solid #e5edf6; }
img { display: block; width: 100%; height: auto; }
</style>
</head>
<body>
<header>
<h1>Nexus UI Visual Gallery</h1>
<p>${items.length} captured route, drawer, modal, overlay, and mobile states.</p>
</header>
<main class="grid">
${items.map((item) => `<figure><figcaption>${escapeHTML(item.title)}</figcaption><img src="${encodeURI(item.file)}" alt="${escapeHTML(item.title)}" /></figure>`).join('\n')}
</main>
</body>
</html>`;
}

function escapeHTML(value) {
    return String(value)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;');
}

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
    if (filePath.endsWith('.png')) {
        return 'image/png';
    }
    return 'application/octet-stream';
}
