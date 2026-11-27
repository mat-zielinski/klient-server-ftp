<script lang="ts">
  // Obraz w galerii: najpierw miniatura m (szybko), potem pełny plik; przybliżanie kółkiem i podwójnym
  // kliknięciem, przesuwanie przeciąganiem przy przybliżeniu.
  import { KROK_PRZYBLIZENIA, MAKS_PRZYBLIZENIE, PRZYBLIZENIE_DWUKLIK } from './stale'

  let { adres, miniatura, nazwa }: { adres: string; miniatura: string; nazwa: string } = $props()

  let skala = $state(1)
  let x = $state(0)
  let y = $state(0)
  let wczytany = $state(false)
  let chwyt: { x: number; y: number; px: number; py: number } | null = null
  let scena = $state<HTMLDivElement | null>(null)

  // kółko z preventDefault wymaga słuchacza, który nie jest pasywny
  $effect(() => {
    const el = scena
    if (!el) return
    el.addEventListener('wheel', kolko, { passive: false })
    return () => el.removeEventListener('wheel', kolko)
  })

  $effect(() => {
    void adres
    skala = 1
    x = 0
    y = 0
    wczytany = false
  })

  function ustawSkale(nowa: number) {
    skala = Math.min(MAKS_PRZYBLIZENIE, Math.max(1, nowa))
    if (skala === 1) {
      x = 0
      y = 0
    }
  }

  function kolko(e: WheelEvent) {
    e.preventDefault()
    ustawSkale(skala * (e.deltaY < 0 ? KROK_PRZYBLIZENIA : 1 / KROK_PRZYBLIZENIA))
  }

  function dwuklik() {
    ustawSkale(skala > 1 ? 1 : PRZYBLIZENIE_DWUKLIK)
  }

  function wcisniecie(e: PointerEvent) {
    if (skala === 1) return
    chwyt = { x: e.clientX, y: e.clientY, px: x, py: y }
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  }

  function ruch(e: PointerEvent) {
    if (!chwyt) return
    x = chwyt.px + (e.clientX - chwyt.x) / skala
    y = chwyt.py + (e.clientY - chwyt.y) / skala
  }

  /** Przesunięcie w jednostkach ekranu jako CSS bez „px” (1 = token --odstep-ramka). */
  const przesuniecie = (n: number) => `calc(var(--odstep-ramka) * ${n.toFixed(1)})`
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="scena"
  class:przyblizony={skala > 1}
  bind:this={scena}
  ondblclick={dwuklik}
  onpointerdown={wcisniecie}
  onpointermove={ruch}
  onpointerup={() => (chwyt = null)}
  onpointercancel={() => (chwyt = null)}
>
  <div class="warstwa" style:transform="scale({skala}) translate({przesuniecie(x)}, {przesuniecie(y)})">
    {#if !wczytany}
      <img class="zastepczy" src={miniatura} alt="" draggable="false" />
    {/if}
    <img class:ukryty={!wczytany} src={adres} alt={nazwa} draggable="false" onload={() => (wczytany = true)} />
  </div>
  {#if skala > 1}
    <span class="skala">{Math.round(skala * 100)}%</span>
  {/if}
</div>

<style>
  .scena {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    min-height: 60vh;
    overflow: hidden;
    touch-action: none;
    cursor: zoom-in;
  }
  .scena.przyblizony {
    cursor: grab;
  }
  .warstwa {
    display: grid;
    place-items: center;
    max-width: 100%;
    max-height: 100%;
    transition: transform var(--czas-szybki);
  }
  .warstwa > img {
    grid-area: 1 / 1;
    max-width: 100%;
    max-height: 70vh;
    object-fit: contain;
    border-radius: var(--promien-segment);
    user-select: none;
  }
  :global(:fullscreen) .warstwa > img {
    max-height: 90vh;
  }
  .zastepczy {
    width: 100%;
    filter: blur(var(--odstep-pozycje));
  }
  .ukryty {
    opacity: 0;
  }
  .skala {
    position: absolute;
    right: var(--odstep-m);
    bottom: var(--odstep-m);
    padding: var(--odstep-pozycje) var(--odstep-s);
    border-radius: var(--promien-kropka);
    background: var(--kolor-panel);
    font-family: var(--czcionka-mono);
    font-size: var(--czcionka-maly);
  }
</style>
