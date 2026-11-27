<script lang="ts">
  // Mapa ciepła dzień tygodnia × godzina (jak „Activity” w UniFi): 7 wierszy (pon..niedz) po 24 komórki,
  // kolor od --kolor-karta (zero) do --kolor-niebieski (maksimum) przez color-mix, dymek z wartością,
  // legenda „Mniej … Więcej”. Brakujące wiersze lub komórki = 0.
  import Dymek from './Dymek.svelte'

  interface Props {
    /** [dzień 0..6 (pon..niedz)][godzina 0..23] */
    komorki: number[][]
    format: (v: number) => string
    /** Nazwa wartości w dymku, np. „Pobrano”. */
    nazwa?: string
  }
  let { komorki, format, nazwa = 'Wartość' }: Props = $props()

  const DNI = ['Pon', 'Wt', 'Śr', 'Czw', 'Pt', 'Sob', 'Niedz']
  const DNI_PELNE = ['Poniedziałek', 'Wtorek', 'Środa', 'Czwartek', 'Piątek', 'Sobota', 'Niedziela']
  const GODZINY = Array.from({ length: 24 }, (_, i) => i)
  const CO_ILE_OPIS = 3
  const STOPNIE_LEGENDY = [0, 0.25, 0.5, 0.75, 1]
  /** Najsłabsza niezerowa komórka ma co najmniej tyle procent barwy (żeby różniła się od zera). */
  const MIN_PROCENT = 12

  let kontener = $state<HTMLDivElement | null>(null)
  let wskazana = $state<{ d: number; g: number; x: number; y: number } | null>(null)

  const wartosc = (d: number, g: number) => Math.max(0, komorki[d]?.[g] ?? 0)
  const max = $derived(komorki.reduce((a, w) => w.reduce((b, v) => Math.max(b, v), a), 0))
  const suma = $derived(komorki.reduce((a, w) => w.reduce((b, v) => b + Math.max(0, v), a), 0))

  function procentBarwy(v: number): number {
    if (!(v > 0) || !(max > 0)) return 0
    return Math.round(MIN_PROCENT + (v / max) * (100 - MIN_PROCENT))
  }
  function tlo(procent: number): string {
    return `color-mix(in srgb, var(--kolor-niebieski) ${procent}%, var(--kolor-karta))`
  }

  function wskaz(e: PointerEvent, d: number, g: number) {
    if (!kontener) return
    const k = kontener.getBoundingClientRect()
    const c = (e.currentTarget as HTMLElement).getBoundingClientRect()
    wskazana = {
      d,
      g,
      x: (c.left + c.width / 2 - k.left) / (k.width || 1),
      y: (c.top - k.top) / (k.height || 1),
    }
  }
  const dwie = (n: number) => String(n).padStart(2, '0')
</script>

<div class="mapa" bind:this={kontener} onpointerleave={() => (wskazana = null)} role="img"
  aria-label="Mapa ciepła: {nazwa.toLowerCase()} wg dnia tygodnia i godziny, razem {format(suma)}">
  <div class="siatka">
    <span></span>
    {#each GODZINY as g (g)}
      <span class="godzina">{g % CO_ILE_OPIS === 0 ? dwie(g) : ''}</span>
    {/each}
    {#each DNI as dzien, d (d)}
      <span class="dzien">{dzien}</span>
      {#each GODZINY as g (g)}
        {@const v = wartosc(d, g)}
        <span class="komorka" class:zero={v <= 0} class:wskazana={wskazana?.d === d && wskazana?.g === g}
          style:background={tlo(procentBarwy(v))} role="presentation" onpointerenter={(e) => wskaz(e, d, g)}></span>
      {/each}
    {/each}
  </div>

  <div class="legenda">
    <span>Mniej</span>
    {#each STOPNIE_LEGENDY as s (s)}
      <span class="probka" style:background={tlo(s === 0 ? 0 : Math.round(MIN_PROCENT + s * (100 - MIN_PROCENT)))}></span>
    {/each}
    <span>Więcej</span>
    <span class="maks">Najwięcej: {format(max)}</span>
  </div>

  {#if wskazana}
    <Dymek x={wskazana.x} y={wskazana.y}>
      <span class="dymek-tytul">{DNI_PELNE[wskazana.d]}, {dwie(wskazana.g)}:00–{dwie((wskazana.g + 1) % 24)}:00</span>
      <span class="dymek-wiersz">
        <span class="dymek-znacznik" style:background="var(--kolor-niebieski)"></span>
        {nazwa}
        <span class="dymek-wartosc">{format(wartosc(wskazana.d, wskazana.g))}</span>
      </span>
    </Dymek>
  {/if}
</div>

<style>
  .mapa {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-3);
    min-width: 0;
  }
  .siatka {
    display: grid;
    grid-template-columns: auto repeat(24, minmax(0, 1fr));
    grid-auto-rows: auto;
    gap: var(--odstep-pozycje);
    align-items: center;
  }
  .godzina,
  .dzien {
    font-family: var(--czcionka-mono);
    font-size: var(--czcionka-drobny);
    color: var(--kolor-opis);
    white-space: nowrap;
  }
  .godzina {
    text-align: left;
    overflow: visible;
  }
  .dzien {
    padding-right: var(--odstep-2);
  }
  .komorka {
    height: var(--wysokosc-znaczek);
    border-radius: var(--promien-kropka);
    outline: var(--odstep-ramka) solid transparent;
    transition: outline-color var(--czas-szybki);
  }
  .komorka.zero {
    box-shadow: inset 0 0 0 var(--odstep-ramka) var(--kolor-siatka);
  }
  .komorka.wskazana {
    outline-color: var(--kolor-tekst);
  }
  .legenda {
    display: flex;
    align-items: center;
    gap: var(--odstep-1);
    flex-wrap: wrap;
    font-size: var(--czcionka-drobny);
    color: var(--kolor-opis);
  }
  .probka {
    width: var(--odstep-5);
    height: var(--odstep-5);
    border-radius: var(--promien-kropka);
    box-shadow: inset 0 0 0 var(--odstep-ramka) var(--kolor-siatka);
  }
  .maks {
    margin-left: auto;
    font-family: var(--czcionka-mono);
  }
</style>
