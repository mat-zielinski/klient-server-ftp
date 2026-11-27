// Generator ikon Skrytki: projekt (sejf ze strzalka pobierania) -> SVG, PNG, ICO.
// Bez bibliotek: wlasny rasteryzator (wielokaty, nadprobkowanie 4x4, gradient liniowy), PNG przez zlib z Node.
// Uruchom z katalogu projektu: node skrypty/ikona.mjs
import { deflateSync } from 'node:zlib';
import { mkdirSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const KORZEN = join(dirname(fileURLToPath(import.meta.url)), '..');
const SIATKA = 512; // jednostki projektu (viewBox 0 0 512 512)
const PROBKI = 4; // nadprobkowanie 4x4 na piksel
const KROK_LUKU = 1.5; // dlugosc odcinka przy splaszczaniu lukow (w jednostkach projektu)
const PROG_MALEJ = 24; // do tego rozmiaru (px) uzywamy uproszczonego symbolu

// ---------- kolory ----------
const hex = (h, a = 1) => [parseInt(h.slice(1, 3), 16), parseInt(h.slice(3, 5), 16), parseInt(h.slice(5, 7), 16), a];
const KOLOR = {
  tloGora: hex('#3d7ef0'),
  tloSrodek: hex('#2856c4'),
  tloDol: hex('#10204a'),
  symbol: hex('#f4f8ff'),
  bialy: hex('#ffffff'),
  granat: hex('#0f1a2e'), // symbol zasobnika na jasny pasek zadan
  zielony: hex('#34d36f'),
  cien: hex('#050c1e', 0.32),
  blask: hex('#ffffff', 0.2),
  blaskZero: hex('#ffffff', 0),
  obwodka: hex('#ffffff', 0.14),
};

// ---------- sciezki (jednoczesnie SVG i wielokaty) ----------
class Sciezka {
  constructor(s = 1, dx = 0, dy = 0) {
    this.s = s; this.dx = dx; this.dy = dy;
    this.d = [];
    this.kontury = [];
  }
  p(x, y) { return [x * this.s + this.dx, y * this.s + this.dy]; }
  // Wielokat o zaokraglonych narozach: punkty [x, y, promien].
  wielokat(punkty) {
    const kontur = [];
    const n = punkty.length;
    punkty.forEach((pk, i) => {
      const [vx, vy] = this.p(pk[0], pk[1]);
      const r = (pk[2] || 0) * this.s;
      const [px, py] = this.p(punkty[(i + n - 1) % n][0], punkty[(i + n - 1) % n][1]);
      const [nx, ny] = this.p(punkty[(i + 1) % n][0], punkty[(i + 1) % n][1]);
      const polecenie = i === 0 ? 'M' : 'L';
      if (r <= 0) {
        kontur.push([vx, vy]);
        this.d.push(`${polecenie}${f(vx)} ${f(vy)}`);
        return;
      }
      this.naroze(kontur, [vx, vy], [px, py], [nx, ny], r, polecenie);
    });
    this.d.push('Z');
    this.kontury.push(kontur);
    return this;
  }
  naroze(kontur, v, pp, nn, r, polecenie) {
    const a = norm([pp[0] - v[0], pp[1] - v[1]]);
    const b = norm([nn[0] - v[0], nn[1] - v[1]]);
    const kat = Math.acos(Math.max(-1, Math.min(1, a[0] * b[0] + a[1] * b[1])));
    const odl = r / Math.tan(kat / 2);
    const t1 = [v[0] + a[0] * odl, v[1] + a[1] * odl];
    const t2 = [v[0] + b[0] * odl, v[1] + b[1] * odl];
    const dw = norm([a[0] + b[0], a[1] + b[1]]);
    const sr = r / Math.sin(kat / 2);
    const c = [v[0] + dw[0] * sr, v[1] + dw[1] * sr];
    const k1 = Math.atan2(t1[1] - c[1], t1[0] - c[0]);
    let delta = Math.atan2(t2[1] - c[1], t2[0] - c[0]) - k1;
    while (delta > Math.PI) delta -= 2 * Math.PI;
    while (delta < -Math.PI) delta += 2 * Math.PI;
    this.d.push(`${polecenie}${f(t1[0])} ${f(t1[1])}A${f(r)} ${f(r)} 0 0 ${delta > 0 ? 1 : 0} ${f(t2[0])} ${f(t2[1])}`);
    const kroki = Math.ceil((Math.abs(delta) * r) / KROK_LUKU) + 2;
    for (let i = 0; i <= kroki; i++) {
      const k = k1 + (delta * i) / kroki;
      kontur.push([c[0] + r * Math.cos(k), c[1] + r * Math.sin(k)]);
    }
  }
  prostokat(x, y, w, h, r) {
    return this.wielokat([[x, y, r], [x + w, y, r], [x + w, y + h, r], [x, y + h, r]]);
  }
  okrag(cx, cy, r) {
    const [x, y] = this.p(cx, cy);
    const rr = r * this.s;
    this.d.push(`M${f(x + rr)} ${f(y)}A${f(rr)} ${f(rr)} 0 1 1 ${f(x - rr)} ${f(y)}A${f(rr)} ${f(rr)} 0 1 1 ${f(x + rr)} ${f(y)}Z`);
    const kroki = Math.ceil((2 * Math.PI * rr) / KROK_LUKU) + 8;
    const kontur = [];
    for (let i = 0; i < kroki; i++) {
      const k = (2 * Math.PI * i) / kroki;
      kontur.push([x + rr * Math.cos(k), y + rr * Math.sin(k)]);
    }
    this.kontury.push(kontur);
    return this;
  }
}
const f = (v) => String(Math.round(v * 100) / 100);
const norm = ([x, y]) => { const l = Math.hypot(x, y) || 1; return [x / l, y / l]; };

// ---------- projekt ----------
// Symbol "duzy": sejf z zawiasami i nozkami, w drzwiach wyciety ksztalt strzalki pobierania nad tacka.
// Symbol "maly" (<= 24 px): bez zawiasow i tacki, wieksza strzalka, zeby byl czytelny w 16x16.
const SYMBOL = {
  duzy: {
    granice: [100, 112, 412, 404],
    korpus: [120, 112, 272, 264, 48],
    dodatki: [[100, 160, 28, 56, 10], [100, 272, 28, 56, 10], [156, 360, 56, 44, 12], [300, 360, 56, 44, 12]],
    strzalka: [[226, 150, 10], [286, 150, 10], [286, 232, 4], [334, 232, 12], [256, 312, 14], [178, 232, 12], [226, 232, 4]],
    tacka: [182, 328, 148, 22, 11],
  },
  maly: {
    granice: [88, 96, 424, 432],
    korpus: [88, 96, 336, 304, 56],
    dodatki: [[128, 384, 72, 48, 14], [312, 384, 72, 48, 14]],
    strzalka: [[216, 136, 12], [296, 136, 12], [296, 236, 4], [358, 236, 14], [256, 348, 16], [154, 236, 14], [216, 236, 4]],
    tacka: null,
  },
};

// Ksztalty symbolu; s, dx, dy przeksztalcaja wspolrzedne projektu.
function symbol(wariant, kolor, s = 1, dx = 0, dy = 0, zCieniem = false) {
  const w = SYMBOL[wariant];
  const ksztalty = [];
  if (zCieniem) {
    const c = new Sciezka(s, dx, dy + 12 * s).prostokat(...w.korpus);
    ksztalty.push({ sciezka: c, wypelnienie: KOLOR.cien });
  }
  const dodatki = new Sciezka(s, dx, dy);
  w.dodatki.forEach((p) => dodatki.prostokat(...p));
  const korpus = new Sciezka(s, dx, dy).prostokat(...w.korpus).wielokat(w.strzalka);
  if (w.tacka) korpus.prostokat(...w.tacka);
  ksztalty.push({ sciezka: dodatki, wypelnienie: kolor });
  ksztalty.push({ sciezka: korpus, wypelnienie: kolor, regula: 'evenodd' });
  return ksztalty;
}

// Dopasowanie symbolu do kwadratu [x0, y0, bok] (srodek granic na srodek kwadratu).
function dopasuj(wariant, x0, y0, bok) {
  const [a, b, c, d] = SYMBOL[wariant].granice;
  const s = bok / Math.max(c - a, d - b);
  return [s, x0 + bok / 2 - ((a + c) / 2) * s, y0 + bok / 2 - ((b + d) / 2) * s];
}

const GRADIENT_TLA = { x1: 96, y1: 16, x2: 400, y2: 500, stopy: [[0, KOLOR.tloGora], [0.5, KOLOR.tloSrodek], [1, KOLOR.tloDol]] };
const GRADIENT_BLASKU = { x1: 256, y1: 0, x2: 256, y2: 300, stopy: [[0, KOLOR.blask], [1, KOLOR.blaskZero]] };

// Pelna kolorowa ikona: zaokraglony kwadrat z gradientem i bialy sejf.
function scenaPelna(rozmiar) {
  const maly = rozmiar <= PROG_MALEJ;
  const m = rozmiar <= 32 ? 8 : 24;
  const r = maly ? 104 : 112;
  const kafel = () => new Sciezka().prostokat(m, m, SIATKA - 2 * m, SIATKA - 2 * m, r);
  const ksztalty = [{ sciezka: kafel(), wypelnienie: GRADIENT_TLA }];
  ksztalty.push({ sciezka: kafel(), wypelnienie: GRADIENT_BLASKU });
  if (!maly) {
    const obw = kafel().prostokat(m + 5, m + 5, SIATKA - 2 * m - 10, SIATKA - 2 * m - 10, r - 5);
    ksztalty.push({ sciezka: obw, wypelnienie: KOLOR.obwodka, regula: 'evenodd' });
  }
  const sym = maly ? symbol('maly', KOLOR.symbol) : symbol('duzy', KOLOR.symbol, 1, 0, 0, true);
  return ksztalty.concat(sym);
}

// Ikona pelnoekranowa (maskowalna PWA, apple-touch): tlo bez zaokraglen, symbol w strefie bezpiecznej.
function scenaPelnoekranowa(bok) {
  const tlo = () => new Sciezka().prostokat(0, 0, SIATKA, SIATKA, 0);
  const [s, dx, dy] = dopasuj('duzy', (SIATKA - bok) / 2, (SIATKA - bok) / 2, bok);
  return [
    { sciezka: tlo(), wypelnienie: GRADIENT_TLA },
    { sciezka: tlo(), wypelnienie: GRADIENT_BLASKU },
    ...symbol('duzy', KOLOR.symbol, s, dx, dy, true),
  ];
}

// Ikona zasobnika: symbol na przezroczystym (bialy na ciemny pasek, granatowy na jasny), aktywna z zielona kropka.
function scenaZasobnika(rozmiar, aktywna, kolor = KOLOR.bialy) {
  const wariant = rozmiar <= PROG_MALEJ ? 'maly' : 'duzy';
  const [s, dx, dy] = dopasuj(wariant, 12, 12, SIATKA - 24);
  const ksztalty = symbol(wariant, kolor, s, dx, dy);
  if (aktywna) {
    ksztalty.push({ sciezka: new Sciezka().okrag(404, 404, 132), wypelnienie: KOLOR.bialy, tryb: 'wymaz' });
    ksztalty.push({ sciezka: new Sciezka().okrag(404, 404, 96), wypelnienie: KOLOR.zielony });
  }
  return ksztalty;
}

// ---------- rasteryzator ----------
function rasteryzuj(ksztalty, rozmiar) {
  const bufor = new Float64Array(rozmiar * rozmiar * 4); // RGBA z alfa premnozona, 0..1
  const skala = rozmiar / SIATKA;
  for (const k of ksztalty) {
    const pokrycie = pokrycieKsztaltu(k, rozmiar, skala);
    for (let i = 0; i < rozmiar * rozmiar; i++) {
      const c = pokrycie[i];
      if (c === 0) continue;
      const o = i * 4;
      if (k.tryb === 'wymaz') {
        for (let j = 0; j < 4; j++) bufor[o + j] *= 1 - c;
        continue;
      }
      const [r, g, b, a] = kolorW(k.wypelnienie, ((i % rozmiar) + 0.5) / skala, (Math.floor(i / rozmiar) + 0.5) / skala);
      const al = a * c;
      bufor[o] = (r / 255) * al + bufor[o] * (1 - al);
      bufor[o + 1] = (g / 255) * al + bufor[o + 1] * (1 - al);
      bufor[o + 2] = (b / 255) * al + bufor[o + 2] * (1 - al);
      bufor[o + 3] = al + bufor[o + 3] * (1 - al);
    }
  }
  const wynik = Buffer.alloc(rozmiar * rozmiar * 4);
  for (let i = 0; i < rozmiar * rozmiar; i++) {
    const a = bufor[i * 4 + 3];
    for (let j = 0; j < 3; j++) wynik[i * 4 + j] = a > 0 ? Math.round(Math.min(1, bufor[i * 4 + j] / a) * 255) : 0;
    wynik[i * 4 + 3] = Math.round(a * 255);
  }
  return wynik;
}

// Pokrycie pikseli (0..1) wielokatami ksztaltu: linie skanowania w podpikselach, regula nonzero albo evenodd.
function pokrycieKsztaltu(k, rozmiar, skala) {
  const pokrycie = new Float64Array(rozmiar * rozmiar);
  const krawedzie = [];
  for (const kontur of k.sciezka.kontury) {
    for (let i = 0; i < kontur.length; i++) {
      const a = kontur[i];
      const b = kontur[(i + 1) % kontur.length];
      if (a[1] !== b[1]) krawedzie.push([a[0] * skala, a[1] * skala, b[0] * skala, b[1] * skala]);
    }
  }
  const waga = 1 / (PROBKI * PROBKI);
  const evenodd = k.regula === 'evenodd';
  for (let py = 0; py < rozmiar; py++) {
    for (let sy = 0; sy < PROBKI; sy++) {
      const y = py + (sy + 0.5) / PROBKI;
      const przeciecia = [];
      for (const [x1, y1, x2, y2] of krawedzie) {
        if ((y >= y1 && y < y2) || (y >= y2 && y < y1)) {
          przeciecia.push([x1 + ((y - y1) * (x2 - x1)) / (y2 - y1), y2 > y1 ? 1 : -1]);
        }
      }
      if (przeciecia.length === 0) continue;
      przeciecia.sort((p, q) => p[0] - q[0]);
      let nawiniecie = 0;
      for (let i = 0; i < przeciecia.length - 1; i++) {
        nawiniecie += przeciecia[i][1];
        const wewnatrz = evenodd ? (i + 1) % 2 === 1 : nawiniecie !== 0;
        if (wewnatrz) dodajOdcinek(pokrycie, py * rozmiar, rozmiar, przeciecia[i][0], przeciecia[i + 1][0], waga);
      }
    }
  }
  return pokrycie;
}

function dodajOdcinek(pokrycie, wiersz, rozmiar, xa, xb, waga) {
  const start = Math.max(0, Math.floor(xa));
  const koniec = Math.min(rozmiar - 1, Math.floor(xb));
  for (let px = start; px <= koniec; px++) {
    for (let sx = 0; sx < PROBKI; sx++) {
      const x = px + (sx + 0.5) / PROBKI;
      if (x >= xa && x < xb) pokrycie[wiersz + px] += waga;
    }
  }
}

function kolorW(w, x, y) {
  if (!w.stopy) return w;
  const dx = w.x2 - w.x1;
  const dy = w.y2 - w.y1;
  const t = Math.max(0, Math.min(1, ((x - w.x1) * dx + (y - w.y1) * dy) / (dx * dx + dy * dy)));
  for (let i = 1; i < w.stopy.length; i++) {
    const [t1, k1] = w.stopy[i];
    const [t0, k0] = w.stopy[i - 1];
    if (t <= t1) {
      const u = t1 === t0 ? 0 : (t - t0) / (t1 - t0);
      return k0.map((v, j) => v + (k1[j] - v) * u);
    }
  }
  return w.stopy[w.stopy.length - 1][1];
}

// ---------- PNG i ICO ----------
const TABLICA_CRC = Array.from({ length: 256 }, (_, n) => {
  let c = n;
  for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
  return c >>> 0;
});
function crc32(buf) {
  let c = 0xffffffff;
  for (const b of buf) c = TABLICA_CRC[(c ^ b) & 0xff] ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
}
function blok(typ, dane) {
  const dl = Buffer.alloc(4);
  dl.writeUInt32BE(dane.length);
  const td = Buffer.concat([Buffer.from(typ, 'ascii'), dane]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(td));
  return Buffer.concat([dl, td, crc]);
}
function png(rgba, rozmiar) {
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(rozmiar, 0);
  ihdr.writeUInt32BE(rozmiar, 4);
  ihdr[8] = 8; // glebia bitowa
  ihdr[9] = 6; // RGBA
  const surowe = Buffer.alloc(rozmiar * (rozmiar * 4 + 1));
  for (let y = 0; y < rozmiar; y++) rgba.copy(surowe, y * (rozmiar * 4 + 1) + 1, y * rozmiar * 4, (y + 1) * rozmiar * 4);
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    blok('IHDR', ihdr),
    blok('IDAT', deflateSync(surowe, { level: 9 })),
    blok('IEND', Buffer.alloc(0)),
  ]);
}
// ICO z osadzonymi PNG (obslugiwane od Windows Vista).
function ico(obrazy) {
  const naglowek = Buffer.alloc(6 + 16 * obrazy.length);
  naglowek.writeUInt16LE(1, 2);
  naglowek.writeUInt16LE(obrazy.length, 4);
  let przesuniecie = naglowek.length;
  obrazy.forEach(({ rozmiar, dane }, i) => {
    const o = 6 + 16 * i;
    naglowek[o] = rozmiar >= 256 ? 0 : rozmiar;
    naglowek[o + 1] = rozmiar >= 256 ? 0 : rozmiar;
    naglowek.writeUInt16LE(1, o + 4); // plaszczyzny
    naglowek.writeUInt16LE(32, o + 6); // bity na piksel
    naglowek.writeUInt32LE(dane.length, o + 8);
    naglowek.writeUInt32LE(przesuniecie, o + 12);
    przesuniecie += dane.length;
  });
  return Buffer.concat([naglowek, ...obrazy.map((o) => o.dane)]);
}

// ---------- SVG ----------
function svg(ksztalty, tytul) {
  const defs = [];
  const tresc = [];
  let licznik = 0;
  const farba = (w) => {
    if (!w.stopy) return `fill="${kolorSvg(w)}"${w[3] < 1 ? ` fill-opacity="${f(w[3])}"` : ''}`;
    const id = `g${++licznik}`;
    const stopy = w.stopy.map(([t, k]) => `<stop offset="${f(t)}" stop-color="${kolorSvg(k)}"${k[3] < 1 ? ` stop-opacity="${f(k[3])}"` : ''}/>`).join('');
    defs.push(`<linearGradient id="${id}" gradientUnits="userSpaceOnUse" x1="${w.x1}" y1="${w.y1}" x2="${w.x2}" y2="${w.y2}">${stopy}</linearGradient>`);
    return `fill="url(#${id})"`;
  };
  for (const k of ksztalty) {
    const d = k.sciezka.d.join('');
    if (k.tryb === 'wymaz') {
      const id = `m${++licznik}`;
      defs.push(`<mask id="${id}"><rect width="512" height="512" fill="#fff"/><path d="${d}" fill="#000"/></mask>`);
      tresc.splice(0, tresc.length, `<g mask="url(#${id})">${tresc.join('')}</g>`);
      continue;
    }
    tresc.push(`<path d="${d}" ${farba(k.wypelnienie)}${k.regula === 'evenodd' ? ' fill-rule="evenodd"' : ''}/>`);
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" width="512" height="512">\n<title>${tytul}</title>\n<defs>${defs.join('')}</defs>\n${tresc.join('\n')}\n</svg>\n`;
}
const kolorSvg = ([r, g, b]) => '#' + [r, g, b].map((v) => Math.round(v).toString(16).padStart(2, '0')).join('');

// Wersja do interfejsu: kolory z tokenow (var(--...)), bez wartosci szesnastkowych.
function logoInterfejsu() {
  const sym = symbol('duzy', null);
  const kafel = new Sciezka().prostokat(24, 24, 464, 464, 112).d.join('');
  const zawiasy = sym[0].sciezka.d.join('');
  const korpus = sym[1].sciezka.d.join('');
  return [
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" aria-hidden="true">',
    '<defs><linearGradient id="skrytka-logo-tlo" gradientUnits="userSpaceOnUse" x1="96" y1="16" x2="400" y2="500">',
    '<stop offset="0" style="stop-color:var(--kolor-niebieski)"/><stop offset="1" style="stop-color:var(--kolor-niebieski-tlo)"/></linearGradient>',
    '<linearGradient id="skrytka-logo-cien" gradientUnits="userSpaceOnUse" x1="256" y1="160" x2="256" y2="500">',
    '<stop offset="0" style="stop-color:var(--kolor-cien);stop-opacity:0"/><stop offset="1" style="stop-color:var(--kolor-cien)"/></linearGradient></defs>',
    `<path d="${kafel}" fill="url(#skrytka-logo-tlo)"/>`,
    `<path d="${kafel}" fill="url(#skrytka-logo-cien)"/>`,
    `<path d="${zawiasy}" style="fill:var(--kolor-na-ciemnym)"/>`,
    `<path d="${korpus}" fill-rule="evenodd" style="fill:var(--kolor-na-ciemnym)"/>`,
    '</svg>',
  ].join('');
}

// ---------- zapis ----------
function zapisz(sciezka, dane) {
  const pelna = join(KORZEN, sciezka);
  mkdirSync(dirname(pelna), { recursive: true });
  writeFileSync(pelna, dane);
  console.log(`zapisano ${sciezka} (${dane.length} B)`);
}
const obraz = (scena, rozmiar) => png(rasteryzuj(scena(rozmiar), rozmiar), rozmiar);
const zestawIco = (scena, rozmiary) => ico(rozmiary.map((r) => ({ rozmiar: r, dane: obraz(scena, r) })));

const pelna = (r) => scenaPelna(r);
const maskowalna = () => scenaPelnoekranowa(272);
const apple = () => scenaPelnoekranowa(330);
const zasobnik = (r) => scenaZasobnika(r, false);
const zasobnikAktywny = (r) => scenaZasobnika(r, true);
const zasobnikCiemny = (r) => scenaZasobnika(r, false, KOLOR.granat);
const zasobnikCiemnyAktywny = (r) => scenaZasobnika(r, true, KOLOR.granat);
const ROZMIARY_APLIKACJI = [16, 24, 32, 48, 64, 128, 256];
const ROZMIARY_ZASOBNIKA = [16, 20, 24, 32, 48];

zapisz('assets/ikona/skrytka.svg', svg(pelna(256), 'Skrytka'));
zapisz('assets/ikona/skrytka-mala.svg', svg(pelna(16), 'Skrytka (16-24 px)'));
zapisz('assets/ikona/skrytka-maskowalna.svg', svg(maskowalna(), 'Skrytka (maskowalna)'));
zapisz('assets/ikona/zasobnik.svg', svg(zasobnik(32), 'Skrytka (zasobnik)'));
zapisz('assets/ikona/zasobnik-aktywny.svg', svg(zasobnikAktywny(32), 'Skrytka (zasobnik, transfer)'));
zapisz('assets/ikona/zasobnik-ciemny.svg', svg(zasobnikCiemny(32), 'Skrytka (zasobnik, jasny pasek)'));
zapisz('assets/ikona/zasobnik-ciemny-aktywny.svg', svg(zasobnikCiemnyAktywny(32), 'Skrytka (zasobnik, jasny pasek, transfer)'));

const png256 = obraz(pelna, 256);
const icoAplikacji = zestawIco(pelna, ROZMIARY_APLIKACJI);
const icoZasobnika = zestawIco(zasobnik, ROZMIARY_ZASOBNIKA);
const icoAktywnego = zestawIco(zasobnikAktywny, ROZMIARY_ZASOBNIKA);
const icoCiemnego = zestawIco(zasobnikCiemny, ROZMIARY_ZASOBNIKA);
const icoCiemnegoAktywnego = zestawIco(zasobnikCiemnyAktywny, ROZMIARY_ZASOBNIKA);
zapisz('assets/ikona/skrytka-256.png', png256);
zapisz('assets/ikona/skrytka.ico', icoAplikacji);
zapisz('assets/ikona/zasobnik.ico', icoZasobnika);
zapisz('assets/ikona/zasobnik-aktywny.ico', icoAktywnego);
zapisz('assets/ikona/zasobnik-ciemny.ico', icoCiemnego);
zapisz('assets/ikona/zasobnik-ciemny-aktywny.ico', icoCiemnegoAktywnego);
zapisz('assets/linux/skrytka.png', png256);

zapisz('wewn/aplikacja/ikony/skrytka.png', png256);
zapisz('wewn/aplikacja/ikony/zasobnik.ico', icoZasobnika);
zapisz('wewn/aplikacja/ikony/zasobnik-aktywny.ico', icoAktywnego);
zapisz('wewn/aplikacja/ikony/zasobnik-ciemny.ico', icoCiemnego);
zapisz('wewn/aplikacja/ikony/zasobnik-ciemny-aktywny.ico', icoCiemnegoAktywnego);

zapisz('web/public/ikona-32.png', obraz(pelna, 32));
zapisz('web/public/ikona-192.png', obraz(pelna, 192));
zapisz('web/public/ikona-512.png', obraz(pelna, 512));
zapisz('web/public/ikona-maskowalna-512.png', obraz(maskowalna, 512));
zapisz('web/public/apple-touch-icon.png', obraz(apple, 180));

const LOGO = `// Wygenerowane przez skrypty/ikona.mjs (nie edytuj recznie).
// Logo Skrytki do interfejsu: kolory z tokenow (var(--...)), dopasowuje sie do motywu.
export const LOGO_SVG: string =
  ${JSON.stringify(logoInterfejsu())};
`;
zapisz('web/src/ikony/logo.ts', LOGO);

// Podglad do recznej kontroli (nie trafia do projektu): node skrypty/ikona.mjs --podglad <katalog>
const ip = process.argv.indexOf('--podglad');
if (ip > 0) {
  const kat = process.argv[ip + 1];
  mkdirSync(kat, { recursive: true });
  const warianty = { pelna, zasobnik, zasobnikAktywny, zasobnikCiemny, zasobnikCiemnyAktywny, maskowalna, apple };
  for (const [nazwa, scena] of Object.entries(warianty)) {
    for (const r of [16, 20, 24, 32, 48, 256]) writeFileSync(join(kat, `${nazwa}-${r}.png`), obraz(scena, r));
  }
}
