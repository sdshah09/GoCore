const puppeteer = require('puppeteer');
const path = require('path');
const fs = require('fs');

(async () => {
    console.log('🚀 Starting PDF generation (5 slides)...');

    const browser = await puppeteer.launch({
        headless: 'new',
        args: ['--no-sandbox', '--disable-setuid-sandbox']
    });

    const page = await browser.newPage();
    await page.setViewport({ width: 1080, height: 1080, deviceScaleFactor: 2 });

    const htmlPath = path.resolve(__dirname, 'index.html');
    await page.goto(`file://${htmlPath}`, { waitUntil: 'networkidle0' });

    console.log('📄 Page loaded. Capturing slides...');

    const slideImages = [];

    for (let i = 1; i <= 5; i++) {
        console.log(`  📸 Capturing slide ${i}...`);
        await page.evaluate((slideNum) => { showSlide(slideNum); }, i);
        await new Promise(r => setTimeout(r, 500));

        const slideEl = await page.$(`#slide-${i}`);
        const imgBuffer = await slideEl.screenshot({ type: 'png', omitBackground: false });
        slideImages.push(imgBuffer);

        const pngPath = path.resolve(__dirname, `slide-${i}.png`);
        fs.writeFileSync(pngPath, imgBuffer);
        console.log(`  ✅ Saved ${pngPath}`);
    }

    console.log('\n📑 Generating combined PDF...');

    const pdfPage = await browser.newPage();
    await pdfPage.setViewport({ width: 1080, height: 5400, deviceScaleFactor: 2 });

    const imagesHtml = slideImages.map((buf) => {
        const b64 = buf.toString('base64');
        return `<div class="pdf-slide"><img src="data:image/png;base64,${b64}" /></div>`;
    }).join('\n');

    const pdfHtml = `
    <!DOCTYPE html>
    <html>
    <head>
        <style>
            * { margin: 0; padding: 0; box-sizing: border-box; }
            body { background: #0a0a0f; }
            .pdf-slide {
                width: 1080px;
                height: 1080px;
                page-break-after: always;
                page-break-inside: avoid;
                overflow: hidden;
            }
            .pdf-slide:last-child { page-break-after: auto; }
            .pdf-slide img { width: 1080px; height: 1080px; display: block; }
        </style>
    </head>
    <body>${imagesHtml}</body>
    </html>`;

    await pdfPage.setContent(pdfHtml, { waitUntil: 'networkidle0' });

    const pdfPath = path.resolve(__dirname, 'helm-how-it-works.pdf');
    await pdfPage.pdf({
        path: pdfPath,
        width: '1080px',
        height: '1080px',
        printBackground: true,
        margin: { top: 0, right: 0, bottom: 0, left: 0 },
        preferCSSPageSize: false
    });

    console.log(`\n✅ PDF saved to: ${pdfPath}`);
    console.log(`✅ Individual PNGs: slide-1.png through slide-5.png`);

    await browser.close();
    console.log('\n🎉 Done! Your 5-slide carousel is ready for LinkedIn.');
})();
