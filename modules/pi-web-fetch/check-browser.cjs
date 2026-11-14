// Verify Puppeteer from the actual consuming package, with its sandbox enabled.
const { createRequire } = require("node:module");
const { resolve } = require("node:path");
const packagePath = resolve(process.argv[2]);
require("node:fs").accessSync(packagePath);
const requireFrom = createRequire(packagePath);
(async () => {
    const browser = await requireFrom("puppeteer").launch({ headless: true });
    try {
        const page = await browser.newPage();
        await page.setContent("<html><body><h1>vbx-browser-ok</h1></body></html>");
        if (await page.$eval("h1", el => el.textContent) !== "vbx-browser-ok") {
            throw new Error("Browser rendering check failed");
        }
    } finally {
        await browser.close();
    }
})().catch(error => { console.error(error.message); process.exitCode = 1; });
