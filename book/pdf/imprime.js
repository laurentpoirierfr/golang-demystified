// Imprime le livre en PDF : sert le site construit dans l'environnement
// « pdf » (site/config/pdf), ouvre la page /livre/ dans Chromium, attend que
// Paged.js l'ait paginée, puis l'imprime. Lancé par make book, dans l'image
// zenika/alpine-chrome:*-with-puppeteer :
//
//   /site    le site construit (hugo --environment pdf), avec les polices
//            (polices/) et paged.polyfill.min.js
//   /sortie  où écrire golang-demystified.pdf
const fs = require("fs");
const http = require("http");
const path = require("path");
const puppeteer = require("puppeteer");

const PORT = 8123;
const types = {
  ".html": "text/html; charset=utf-8", ".css": "text/css", ".js": "text/javascript",
  ".svg": "image/svg+xml", ".png": "image/png", ".otf": "font/otf", ".ttf": "font/ttf",
};

function fichier(url) {
  const p = decodeURIComponent(new URL(url, "http://x").pathname);
  const f = path.join("/site", path.normalize(p));
  return p.endsWith("/") ? path.join(f, "index.html") : f;
}

const serveur = http.createServer((req, res) => {
  const f = fichier(req.url);
  fs.readFile(f, (err, data) => {
    if (err) {
      res.writeHead(404).end();
      return;
    }
    res.writeHead(200, { "Content-Type": types[path.extname(f)] || "application/octet-stream" });
    res.end(data);
  });
});

(async () => {
  await new Promise((ok) => serveur.listen(PORT, ok));
  const navigateur = await puppeteer.launch({
    executablePath: "/usr/bin/chromium-browser",
    args: ["--no-sandbox", "--disable-gpu", "--font-render-hinting=none"],
  });
  try {
    const page = await navigateur.newPage();
    page.on("console", (m) => { if (m.type() === "error") console.error("navigateur :", m.text()); });
    page.on("pageerror", (e) => console.error("navigateur :", e.message));
    page.on("requestfailed", (r) => console.error("introuvable :", r.url()));
    page.on("response", (r) => { if (r.status() >= 400) console.error("introuvable :", r.url()); });

    const debut = Date.now();
    await page.goto(`http://localhost:${PORT}/livre/`, { waitUntil: "load" });
    await page.waitForFunction("window.livrePagine === true", { timeout: 15 * 60 * 1000, polling: 1000 });
    const pages = await page.evaluate(() => document.querySelectorAll(".pagedjs_page").length);
    console.log(`livre paginé : ${pages} pages en ${Math.round((Date.now() - debut) / 1000)} s`);

    await page.pdf({
      path: "/sortie/golang-demystified.pdf",
      preferCSSPageSize: true,
      printBackground: true,
      timeout: 15 * 60 * 1000,
    });
    console.log("écrit : golang-demystified.pdf");
  } finally {
    await navigateur.close();
    serveur.close();
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
