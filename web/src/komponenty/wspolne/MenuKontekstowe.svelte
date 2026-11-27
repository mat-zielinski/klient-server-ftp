<script lang="ts" module>
  import type { NazwaIkony } from '../../ikony/ikony'

  export interface PozycjaMenu {
    etykieta: string
    ikona?: NazwaIkony
    /** Drugi wiersz (np. powód wyszarzenia). */
    opis?: string
    onclick?: () => void
    wylaczona?: boolean
    podmenu?: PozycjaMenu[]
    /** Kreska oddzielająca PRZED tą pozycją. */
    separator?: boolean
    niebezpieczna?: boolean
  }
</script>

<script lang="ts">
  // Menu kontekstowe w punkcie (x, y) okna; samo mieści się w oknie, podmenu rozwija się w bok po najechaniu.
  // Zamyka: klik poza menu, Esc, przewinięcie, zmiana rozmiaru okna, wybór pozycji.
  import { onMount } from 'svelte'
  import Ikona from './Ikona.svelte'

  interface Props {
    x: number
    y: number
    pozycje: PozycjaMenu[]
    zamknij: () => void
  }

  let { x, y, pozycje, zamknij }: Props = $props()

  let menu: HTMLDivElement | undefined = $state()
  let lewo = $state(0)
  let gora = $state(0)
  let podmenuWLewo = $state(false)
  let otwarte = $state<number | null>(null)

  $effect(() => {
    lewo = x
    gora = y
    if (!menu) return
    const r = menu.getBoundingClientRect()
    const margines = parseFloat(getComputedStyle(menu).getPropertyValue('--odstep-m')) || 0
    if (x + r.width > innerWidth - margines) lewo = Math.max(margines, innerWidth - r.width - margines)
    if (y + r.height > innerHeight - margines) gora = Math.max(margines, innerHeight - r.height - margines)
    podmenuWLewo = lewo + r.width * 2 > innerWidth - margines
  })

  onMount(() => {
    const poza = (e: Event) => {
      if (menu && !menu.contains(e.target as Node)) zamknij()
    }
    const klawisz = (e: KeyboardEvent) => {
      if (e.key === 'Escape') zamknij()
      else if (e.key === 'ArrowDown' || e.key === 'ArrowUp') przesunFokus(e.key === 'ArrowDown' ? 1 : -1, e)
    }
    const zamknijTeraz = () => zamknij()
    // pozwala zakończyć kliknięcie, które otworzyło menu
    setTimeout(() => {
      addEventListener('pointerdown', poza, true)
      addEventListener('contextmenu', poza, true)
    })
    addEventListener('keydown', klawisz)
    addEventListener('resize', zamknijTeraz)
    addEventListener('scroll', zamknijTeraz, true)
    menu?.querySelector<HTMLButtonElement>('button:not(:disabled)')?.focus()
    return () => {
      removeEventListener('pointerdown', poza, true)
      removeEventListener('contextmenu', poza, true)
      removeEventListener('keydown', klawisz)
      removeEventListener('resize', zamknijTeraz)
      removeEventListener('scroll', zamknijTeraz, true)
    }
  })

  function przesunFokus(krok: number, e: KeyboardEvent) {
    if (!menu) return
    e.preventDefault()
    const przyciski = [...menu.querySelectorAll<HTMLButtonElement>('button:not(:disabled)')].filter(
      (b) => b.offsetParent !== null,
    )
    if (!przyciski.length) return
    const i = przyciski.indexOf(document.activeElement as HTMLButtonElement)
    przyciski[(i + krok + przyciski.length) % przyciski.length].focus()
  }

  function wybierz(p: PozycjaMenu) {
    if (p.wylaczona || p.podmenu) return
    zamknij()
    p.onclick?.()
  }
</script>

{#snippet lista(elementy: PozycjaMenu[], poziom: number)}
  {#each elementy as p, i (i)}
    {#if p.separator}<div class="kreska" role="separator"></div>{/if}
    <div
      class="miejsce"
      role="none"
      onmouseenter={() => poziom === 0 && (otwarte = p.podmenu ? i : null)}
    >
      <button
        type="button"
        role="menuitem"
        class="pozycja"
        class:niebezpieczna={p.niebezpieczna}
        disabled={p.wylaczona}
        aria-haspopup={p.podmenu ? 'menu' : undefined}
        title={p.wylaczona && p.opis ? p.opis : undefined}
        onclick={() => (p.podmenu ? (otwarte = otwarte === i ? null : i) : wybierz(p))}
        onkeydown={(e) => p.podmenu && e.key === 'ArrowRight' && (otwarte = i)}
      >
        <span class="ikona">{#if p.ikona}<Ikona nazwa={p.ikona} rozmiar="s" />{/if}</span>
        <span class="napisy">
          <span class="skroc">{p.etykieta}</span>
          {#if p.opis}<span class="opis skroc">{p.opis}</span>{/if}
        </span>
        {#if p.podmenu}<span class="strzalka"><Ikona nazwa="w_prawo" rozmiar="s" /></span>{/if}
      </button>
      {#if p.podmenu && (poziom > 0 || otwarte === i)}
        <div class="menu podmenu" class:w-lewo={podmenuWLewo} class:zagniezdzone={poziom > 0} role="menu">
          {@render lista(p.podmenu, poziom + 1)}
        </div>
      {/if}
    </div>
  {/each}
{/snippet}

<div
  bind:this={menu}
  class="menu"
  role="menu"
  tabindex="-1"
  style:left="{lewo}px"
  style:top="{gora}px"
  oncontextmenu={(e) => e.preventDefault()}
>
  {@render lista(pozycje, 0)}
</div>

<style>
  .menu {
    position: fixed;
    z-index: var(--warstwa-menu);
    min-width: var(--uklad-menu-w);
    max-width: calc(var(--uklad-menu-w) * 1.6);
    padding: var(--odstep-xs);
    background: var(--kolor-panel);
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
    box-shadow: 0 var(--odstep-m) var(--odstep-sekcja) var(--kolor-cien);
    outline: 0;
  }
  .podmenu {
    position: absolute;
    left: 100%;
    top: calc(-1 * var(--odstep-xs));
  }
  .podmenu.w-lewo {
    left: auto;
    right: 100%;
  }
  .miejsce {
    position: relative;
  }
  .zagniezdzone {
    display: none;
  }
  .miejsce:hover > .zagniezdzone,
  .miejsce:focus-within > .zagniezdzone {
    display: block;
  }
  .pozycja {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    width: 100%;
    min-height: var(--wysokosc-przycisk-maly);
    padding: var(--odstep-xs) var(--odstep-przycisk-maly);
    border: 0;
    border-radius: var(--promien-segment);
    background: transparent;
    color: var(--kolor-tekst);
    font-size: var(--czcionka-maly);
    text-align: left;
  }
  .pozycja:hover:not(:disabled),
  .pozycja:focus-visible {
    background: var(--kolor-aktywny);
    color: var(--kolor-niebieski);
    outline: 0;
  }
  .pozycja:disabled {
    color: var(--kolor-wylaczony-napis);
  }
  .niebezpieczna:not(:disabled) {
    color: var(--kolor-czerwony);
  }
  .ikona {
    display: inline-flex;
    width: var(--uklad-ikona-s);
    flex: none;
  }
  .napisy {
    display: flex;
    flex-direction: column;
    min-width: 0;
    flex: 1;
  }
  .opis {
    font-size: var(--czcionka-drobny);
  }
  .strzalka {
    display: inline-flex;
    color: var(--kolor-opis);
  }
  .kreska {
    height: var(--odstep-ramka);
    margin: var(--odstep-xs) var(--odstep-s);
    background: var(--kolor-siatka);
  }
</style>
