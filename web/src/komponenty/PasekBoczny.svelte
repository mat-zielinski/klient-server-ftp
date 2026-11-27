<script lang="ts" module>
  import type { NazwaIkony } from '../ikony/ikony'

  export interface PozycjaPaska {
    klucz: string
    etykieta: string
    ikona: NazwaIkony
    /** Dymek: do czego służy strona. */
    opis?: string
  }
</script>

<script lang="ts">
  // Pasek boczny jak w stacji (okno.py: PasekBoczny, arkusz.py: _pasek_boczny): u góry nazwa i wersja, pozycje stron
  // (ikona + podpis, zaznaczona tłem `aktywny`, napis i ikona niebieskie), na dole użytkownik, motyw, zwinięcie
  // i Wyloguj (przygaszone). Zwinięty = same ikony z dymkiem; na telefonie wysuwana szuflada.
  import { motyw, NAZWY_MOTYWOW } from '../stan/motyw.svelte'
  import { pasek } from '../stan/pasek.svelte'
  import { uklad } from '../stan/uklad.svelte'
  import Ikona from './wspolne/Ikona.svelte'

  interface Props {
    nazwa: string
    wersja: string
    pozycje: PozycjaPaska[]
    aktywna: string
    login: string
    admin: boolean
    naWybierz: (klucz: string) => void
    naWyloguj: () => void
  }

  let { nazwa, wersja, pozycje, aktywna, login, admin, naWybierz, naWyloguj }: Props = $props()

  const szuflada = $derived(uklad.telefon)
  /** Same ikony: wybór użytkownika albo wąskie okno (nigdy w szufladzie). */
  const zwiniety = $derived(!szuflada && (pasek.zwiniety || uklad.waski))

  const IKONA_MOTYWU = { auto: 'motyw_auto', ciemny: 'ksiezyc', jasny: 'slonce' } as const

  let dymek = $state<{ tekst: string; gora: number } | null>(null)

  function pokazDymek(e: MouseEvent | FocusEvent, tekst: string) {
    if (!zwiniety) return
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
    dymek = { tekst, gora: r.top + r.height / 2 }
  }

  function ukryjDymek() {
    dymek = null
  }

  function wybierz(klucz: string) {
    dymek = null
    pasek.zamknijSzuflade()
    naWybierz(klucz)
  }

  $effect(() => {
    if (!zwiniety) dymek = null
  })
</script>

{#snippet pozycja(etykieta: string, ikona: NazwaIkony, onclick: () => void, opcje: { aktywna?: boolean; przygaszona?: boolean; opis?: string })}
  <button
    type="button"
    class="pozycja"
    class:aktywna={opcje.aktywna}
    class:przygaszona={opcje.przygaszona}
    aria-current={opcje.aktywna ? 'page' : undefined}
    aria-label={zwiniety ? etykieta : undefined}
    title={!zwiniety && opcje.opis ? opcje.opis : undefined}
    {onclick}
    onmouseenter={(e) => pokazDymek(e, etykieta)}
    onmouseleave={ukryjDymek}
    onfocus={(e) => pokazDymek(e, etykieta)}
    onblur={ukryjDymek}
  >
    <Ikona nazwa={ikona} />
    {#if !zwiniety}<span class="napis skroc">{etykieta}</span>{/if}
  </button>
{/snippet}

{#if szuflada && pasek.szuflada}
  <button type="button" class="zaslona" aria-label="Zamknij menu" onclick={() => pasek.zamknijSzuflade()}></button>
{/if}

<nav
  class="pasek"
  class:zwiniety
  class:szuflada
  class:otwarta={pasek.szuflada}
  aria-label="Menu główne"
>
  <div class="marka" class:zwiniety>
    <span class="logo"><Ikona nazwa="dysk" /></span>
    {#if !zwiniety}
      <span class="nazwy">
        <span class="nazwa skroc">{nazwa}</span>
        <span class="wersja skroc">wersja {wersja}</span>
      </span>
    {/if}
    {#if szuflada}
      <button type="button" class="zamknij" aria-label="Zamknij menu" onclick={() => pasek.zamknijSzuflade()}>
        <Ikona nazwa="zamknij" />
      </button>
    {/if}
  </div>

  <div class="lista">
    {#each pozycje as p (p.klucz)}
      {@render pozycja(p.etykieta, p.ikona, () => wybierz(p.klucz), { aktywna: p.klucz === aktywna, opis: p.opis })}
    {/each}
  </div>

  <div class="kreska"></div>

  <div class="dol">
    <div class="uzytkownik" class:zwiniety title={zwiniety ? login : undefined}>
      <span class="awatar"><Ikona nazwa={admin ? 'tarcza_pusta' : 'uzytkownik'} rozmiar="s" /></span>
      {#if !zwiniety}
        <span class="nazwy">
          <span class="login skroc">{login}</span>
          <span class="wersja">{admin ? 'Administrator' : 'Użytkownik'}</span>
        </span>
      {/if}
    </div>
    {@render pozycja(
      NAZWY_MOTYWOW[motyw.wybor],
      IKONA_MOTYWU[motyw.wybor],
      () => motyw.przelacz(),
      { przygaszona: true },
    )}
    {#if !szuflada && !uklad.waski}
      {@render pozycja(pasek.zwiniety ? 'Rozwiń menu' : 'Zwiń menu', 'pasek_boczny', () => pasek.przelaczZwiniecie(), {
        przygaszona: true,
      })}
    {/if}
    {@render pozycja('Wyloguj', 'wyloguj', () => naWyloguj(), { przygaszona: true })}
  </div>
</nav>

{#if dymek}
  <div class="dymek" role="tooltip" style:top="{dymek.gora}px">{dymek.tekst}</div>
{/if}

<style>
  .pasek {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-pozycje);
    flex: none;
    width: var(--uklad-pasek-w);
    height: 100%;
    padding: var(--odstep-xl) var(--odstep-m);
    background: var(--kolor-pasek);
    border-right: var(--ramka);
    transition: width var(--czas-sredni);
    z-index: var(--warstwa-pasek);
  }
  .pasek.zwiniety {
    width: var(--uklad-pasek-zwiniety-w);
  }
  .pasek.szuflada {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: min(calc(var(--uklad-pasek-w) * 1.4), 85vw);
    transform: translateX(-100%);
    transition: transform var(--czas-sredni);
    z-index: var(--warstwa-okno);
    box-shadow: 0 0 var(--odstep-sekcja) var(--kolor-cien);
  }
  .pasek.szuflada.otwarta {
    transform: none;
  }
  .zaslona {
    position: fixed;
    inset: 0;
    border: 0;
    padding: 0;
    background: var(--kolor-zaslona);
    z-index: var(--warstwa-okno);
  }
  .marka {
    display: flex;
    align-items: center;
    gap: var(--odstep-l);
    padding: 0 var(--odstep-xs) var(--odstep-xl);
    min-height: var(--wysokosc-przycisk-duzy);
  }
  .marka.zwiniety {
    justify-content: center;
    padding-left: 0;
    padding-right: 0;
  }
  .logo {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: none;
    width: var(--wysokosc-przycisk);
    height: var(--wysokosc-przycisk);
    border-radius: var(--promien-przycisk);
    background: var(--kolor-niebieski-tlo);
    color: var(--kolor-na-ciemnym);
  }
  .nazwy {
    display: flex;
    flex-direction: column;
    min-width: 0;
    flex: 1;
  }
  .nazwa {
    font-size: var(--czcionka-logo);
    font-weight: var(--czcionka-gruby);
  }
  .wersja {
    font-size: var(--czcionka-drobny);
    color: var(--kolor-opis);
  }
  .zamknij {
    display: inline-flex;
    border: 0;
    padding: var(--odstep-xs);
    border-radius: var(--promien-przycisk);
    background: transparent;
    color: var(--kolor-opis);
  }
  .lista {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-pozycje);
    flex: 1;
    min-height: var(--wysokosc-pozycja-paska);
    overflow-y: auto;
    overflow-x: hidden;
  }
  .pozycja {
    display: flex;
    align-items: center;
    gap: var(--odstep-l);
    width: 100%;
    height: var(--wysokosc-pozycja-paska);
    flex: none;
    padding: 0 var(--odstep-l);
    border: 0;
    border-radius: var(--promien-przycisk);
    background: transparent;
    color: var(--kolor-tekst);
    font-size: var(--czcionka-tekst);
    text-align: left;
  }
  .zwiniety .pozycja {
    justify-content: center;
    padding: 0;
  }
  .pozycja:hover {
    background: var(--kolor-panel);
  }
  .pozycja.przygaszona {
    color: var(--kolor-opis);
  }
  .pozycja.aktywna {
    background: var(--kolor-aktywny);
    color: var(--kolor-niebieski);
    font-weight: var(--czcionka-polgruby);
  }
  .napis {
    flex: 1;
  }
  .kreska {
    flex: none;
    height: var(--odstep-ramka);
    margin: var(--odstep-s) var(--odstep-xs);
    background: var(--kolor-siatka);
  }
  .dol {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-pozycje);
  }
  .uzytkownik {
    display: flex;
    align-items: center;
    gap: var(--odstep-l);
    padding: var(--odstep-xs) var(--odstep-s) var(--odstep-s);
    min-width: 0;
  }
  .uzytkownik.zwiniety {
    justify-content: center;
    padding-left: 0;
    padding-right: 0;
  }
  .awatar {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: none;
    width: var(--wysokosc-przycisk-maly);
    height: var(--wysokosc-przycisk-maly);
    border-radius: var(--promien-kolo);
    background: var(--kolor-przycisk);
    color: var(--kolor-tekst);
  }
  .login {
    font-size: var(--czcionka-maly);
    font-weight: var(--czcionka-polgruby);
  }
  .dymek {
    position: fixed;
    left: calc(var(--uklad-pasek-zwiniety-w) + var(--odstep-s));
    transform: translateY(-50%);
    z-index: var(--warstwa-menu);
    max-width: var(--uklad-dymek-w);
    padding: var(--odstep-xs) var(--odstep-s);
    background: var(--kolor-panel);
    border: var(--ramka);
    border-radius: var(--promien-segment);
    color: var(--kolor-tekst);
    font-size: var(--czcionka-maly);
    white-space: nowrap;
    pointer-events: none;
    box-shadow: 0 var(--odstep-xs) var(--odstep-m) var(--kolor-cien);
  }
</style>
