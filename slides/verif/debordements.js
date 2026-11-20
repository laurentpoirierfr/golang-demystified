// Vérifie les diapositives générées (make slides-verif) : ouvre chaque
// diaporama dans Chromium et signale les diapositives dont le contenu dépasse
// la zone utile (1280 × 720, marges du thème). Le générateur estime la
// hauteur du contenu ; cette vérification mesure le rendu réel. Elle échoue
// s'il y a un débordement.
const puppeteer = require("puppeteer");
const fs = require("fs");
const path = require("path");

const racine = "/diapos";
const fichiers = [];
(function parcours(d) {
  for (const f of fs.readdirSync(d)) {
    const p = path.join(d, f);
    if (fs.statSync(p).isDirectory()) parcours(p);
    else if (f.endsWith(".html") && f !== "index.html") fichiers.push(p);
  }
})(racine);

(async () => {
  const nav = await puppeteer.launch({ args: ["--no-sandbox"], executablePath: "/usr/bin/chromium-browser" });
  const page = await nav.newPage();
  let total = 0, problemes = [];
  for (const f of fichiers.sort()) {
    await page.goto("file://" + f, { waitUntil: "load" });
    await page.evaluate(() => document.fonts.ready);
    const res = await page.evaluate(() => {
      const out = [];
      document.querySelectorAll("svg > foreignObject > section").forEach((s, i) => {
        const bas = s.offsetHeight - parseFloat(getComputedStyle(s).paddingBottom) + 8;
        const droite = s.offsetWidth - parseFloat(getComputedStyle(s).paddingRight) + 8;
        let maxB = 0, maxD = 0;
        for (const c of s.children) {
          if (c.tagName === "HEADER" || c.tagName === "FOOTER") continue;
          maxB = Math.max(maxB, c.offsetTop + c.offsetHeight);
          maxD = Math.max(maxD, c.offsetLeft + c.scrollWidth);
        }
        if (maxB > bas || maxD > droite) {
          const titre = (s.querySelector("h1,h2") || {}).textContent || "";
          out.push({ n: i + 1, titre, bas: Math.round(maxB - bas), droite: Math.round(maxD - droite) });
        }
      });
      return { n: document.querySelectorAll("svg > foreignObject > section").length, out };
    });
    total += res.n;
    for (const o of res.out) problemes.push(`${path.relative(racine, f)} #${o.n} « ${o.titre} » : bas +${o.bas}px, droite +${o.droite}px`);
  }
  console.log(`${fichiers.length} diaporamas, ${total} diapositives, ${problemes.length} débordements`);
  problemes.forEach((p) => console.log(p));
  await nav.close();
  process.exitCode = problemes.length > 0 ? 1 : 0;
})().catch((e) => { console.error(e); process.exit(1); });
