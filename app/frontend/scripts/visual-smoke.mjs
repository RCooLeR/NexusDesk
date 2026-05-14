import {existsSync, mkdirSync} from 'node:fs';
import path from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const indexPath = path.join(root, 'dist', 'index.html');
const screenshotDir = path.join(root, 'dist', 'smoke');

if (!existsSync(indexPath)) {
    console.log('NexusDesk visual smoke skipped: run npm run build first.');
    process.exit(0);
}

let chromium;
try {
    ({chromium} = await import('playwright'));
} catch {
    console.log('NexusDesk visual smoke skipped: Playwright is not installed.');
    process.exit(0);
}

mkdirSync(screenshotDir, {recursive: true});
const browser = await chromium.launch();
const page = await browser.newPage({viewport: {width: 1440, height: 960}});
await page.goto(pathToFileURL(indexPath).href);
await page.screenshot({path: path.join(screenshotDir, 'desktop.png'), fullPage: false});
await page.setViewportSize({width: 390, height: 844});
await page.screenshot({path: path.join(screenshotDir, 'mobile.png'), fullPage: false});
await browser.close();

console.log('NexusDesk visual smoke captured desktop and mobile screenshots.');
