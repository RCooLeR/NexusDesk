import {existsSync, mkdirSync, writeFileSync} from 'node:fs';
import path from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';

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
const browser = await chromium.launch();
const page = await browser.newPage({viewport: {width: 1440, height: 960}});
await page.goto(pathToFileURL(indexPath).href);
await page.screenshot({path: path.join(screenshotDir, 'desktop.png'), fullPage: false});
await page.screenshot({path: path.join(baselineDir, 'desktop.png'), fullPage: false});
await page.setViewportSize({width: 390, height: 844});
await page.screenshot({path: path.join(screenshotDir, 'mobile.png'), fullPage: false});
await page.screenshot({path: path.join(baselineDir, 'mobile.png'), fullPage: false});
await browser.close();

writeFileSync(path.join(baselineDir, 'manifest.json'), `${JSON.stringify({
    generatedAt: new Date().toISOString(),
    source: 'dist/index.html',
    viewports: ['desktop', 'mobile'],
}, null, 2)}\n`);

console.log('NexusDesk visual smoke captured desktop/mobile screenshots and baseline metadata.');
