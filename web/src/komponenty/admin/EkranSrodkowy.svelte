<script lang="ts">
  // Ekran bez paska bocznego (Logowanie, Kreator, link publiczny) w stylu UniFi / SaaS: tło z poświatą i delikatną
  // siatką, karta z cieniem. Szerokie okno: dwie kolumny (marka z hasłem i trzema zaletami po lewej, formularz po
  // prawej); poniżej progu `waski` tylko logo nad kartą. Prawy górny róg: przełącznik motywu. Stopka: wersja.
  import type { Snippet } from 'svelte'
  import type { NazwaIkony } from '../../ikony/ikony'
  import Logo from '../wspolne/Logo.svelte'
  import { sesja } from '../../stan/sesja.svelte'
  import { uklad } from '../../stan/uklad.svelte'
  import Ikona from '../wspolne/Ikona.svelte'
  import PrzelacznikMotywu from './PrzelacznikMotywu.svelte'

  interface Props {
    /** Podtytuł nad kartą (np. „Zaloguj się, aby kontynuować”). */
    podtytul?: string
    /** Szersza karta (kreator, lista plików folderu). */
    szeroka?: boolean
    /** Kolumna marki na szerokim ekranie (domyślnie tak). */
    marka?: boolean
    children: Snippet
  }

  let { podtytul, szeroka = false, marka = true, children }: Props = $props()

  const ZALETY: { ikona: NazwaIkony; tytul: string; opis: string }[] = [
    {
      ikona: 'pobierz',
      tytul: 'Szybkie pobieranie z wznawianiem',
      opis: 'Przerwane pobieranie rusza od miejsca, w którym stanęło.',
    },
    { ikona: 'tarcza', tytul: 'WireGuard i HTTPS', opis: 'Szyfrowane połączenie w domu i poza nim.' },
    {
      ikona: 'aktywnosc',
      tytul: 'Podgląd transferów na żywo',
      opis: 'Kto, co i z jaką prędkością pobiera, w jednym miejscu.',
    },
  ]

  const nazwa = $derived(sesja.stan?.nazwa || 'Skrytka')
  const wersja = $derived(sesja.stan?.wersja ?? '')
  const dwieKolumny = $derived(marka && !uklad.ponizej('waski'))
</script>

<div class="ekran" class:dwie={dwieKolumny}>
  <div class="gora">
    {#if !dwieKolumny}
      <div class="logo">
        <span class="znak"><Logo /></span>
        <span class="nazwa">{nazwa}</span>
      </div>
    {:else}
      <span></span>
    {/if}
    <PrzelacznikMotywu zwarty={uklad.telefon} />
  </div>

  <div class="tresc">
    {#if dwieKolumny}
      <aside class="marka">
        <div class="logo">
          <span class="znak"><Logo /></span>
          <span class="nazwa">{nazwa}</span>
        </div>
        <h1 class="haslo">Twoje pliki, Twój serwer.</h1>
        <p class="wstep">
          Prywatny serwer plików na tym komputerze. Dostęp przez przeglądarkę i FTPS, bez chmury i bez abonamentu.
        </p>
        <ul class="zalety">
          {#each ZALETY as z (z.tytul)}
            <li>
              <span class="punkt"><Ikona nazwa={z.ikona} /></span>
              <span class="tekst">
                <strong>{z.tytul}</strong>
                <span class="opis">{z.opis}</span>
              </span>
            </li>
          {/each}
        </ul>
      </aside>
    {/if}

    <div class="kolumna" class:szeroka>
      {#if podtytul}<p class="podtytul">{podtytul}</p>{/if}
      <div class="karta">
        {@render children()}
      </div>
    </div>
  </div>

  <footer class="stopka">
    {nazwa}{#if wersja}&nbsp;· wersja {wersja}{/if}
  </footer>
</div>

<style>
  .ekran {
    position: relative;
    isolation: isolate;
    min-height: 100%;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-7);
    padding: var(--odstep-xl) var(--odstep-7);
    border-radius: var(--promien-karta);
    overflow: hidden;
    background:
      radial-gradient(circle var(--poswiata-rozmiar) at 12% 0%, var(--kolor-poswiata-1), transparent 70%),
      radial-gradient(circle var(--poswiata-rozmiar) at 100% 100%, var(--kolor-poswiata-2), transparent 70%),
      var(--kolor-tlo);
  }
  /* siatka tła wygaszana ku brzegom */
  .ekran::before {
    content: '';
    position: absolute;
    inset: 0;
    z-index: -1;
    pointer-events: none;
    background-image:
      linear-gradient(var(--kolor-siatka-tla) var(--odstep-ramka), transparent var(--odstep-ramka)),
      linear-gradient(90deg, var(--kolor-siatka-tla) var(--odstep-ramka), transparent var(--odstep-ramka));
    background-size: var(--siatka-tla-rozmiar) var(--siatka-tla-rozmiar);
    mask-image: radial-gradient(ellipse at 50% 30%, var(--kolor-tekst), transparent 75%);
  }
  .gora {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--odstep-m);
  }
  .tresc {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--uklad-wejscie-odstep);
    width: 100%;
  }
  .logo {
    display: flex;
    align-items: center;
    gap: var(--odstep-l);
  }
  .znak {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--wysokosc-przycisk-duzy);
    height: var(--wysokosc-przycisk-duzy);
    border-radius: var(--promien-kafel);
    box-shadow: 0 var(--odstep-xs) var(--odstep-xl) var(--kolor-poswiata-1);
  }
  /* logo aplikacji (ikony/logo.ts, kolory z tokenów) wypełnia znak */
  .znak :global(svg) {
    width: 100%;
    height: 100%;
  }
  .nazwa {
    font-size: var(--czcionka-tytul-strony);
    font-weight: var(--czcionka-gruby);
    letter-spacing: -0.01em;
  }

  /* kolumna marki */
  .marka {
    flex: 0 1 var(--uklad-wejscie-marka-w);
    display: flex;
    flex-direction: column;
    gap: var(--odstep-7);
    animation: wejscie var(--czas-krok) var(--krzywa-wejscia) both;
  }
  .marka .znak {
    width: var(--wysokosc-znak-marki);
    height: var(--wysokosc-znak-marki);
  }
  .haslo {
    margin: var(--odstep-xl) 0 0;
    font-size: var(--czcionka-haslo-marki);
    font-weight: var(--czcionka-gruby);
    line-height: 1.15;
    letter-spacing: -0.02em;
    background: linear-gradient(90deg, var(--kolor-tekst), var(--kolor-niebieski));
    background-clip: text;
    -webkit-background-clip: text;
    color: transparent;
  }
  .wstep {
    margin: 0;
    color: var(--kolor-opis);
    font-size: var(--czcionka-tekst);
    line-height: 1.6;
  }
  .zalety {
    list-style: none;
    margin: var(--odstep-m) 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-7);
  }
  .zalety li {
    display: flex;
    align-items: flex-start;
    gap: var(--odstep-xl);
  }
  .punkt {
    flex: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--wysokosc-punkt-marki);
    height: var(--wysokosc-punkt-marki);
    border-radius: var(--promien-kafel);
    background: var(--kolor-tlo-info);
    border: var(--odstep-ramka) solid var(--kolor-ramka-wejscia);
    color: var(--kolor-niebieski);
  }
  .tekst {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-pozycje);
    min-width: 0;
  }
  .tekst strong {
    font-weight: var(--czcionka-polgruby);
  }

  /* kolumna formularza */
  .kolumna {
    flex: 0 1 var(--uklad-wejscie-w);
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-l);
  }
  .kolumna.szeroka {
    flex-basis: var(--uklad-wejscie-szerokie-w);
  }
  .ekran:not(.dwie) .kolumna {
    flex-grow: 1;
    max-width: var(--uklad-wejscie-w);
  }
  .ekran:not(.dwie) .kolumna.szeroka {
    max-width: var(--uklad-wejscie-szerokie-w);
  }
  .podtytul {
    margin: 0;
    color: var(--kolor-opis);
    text-align: center;
  }
  .karta {
    width: 100%;
    background: var(--kolor-karta-wejscia);
    border: var(--odstep-ramka) solid var(--kolor-ramka-wejscia);
    border-radius: var(--promien-karta);
    box-shadow: var(--cien-karta-wejscia);
    backdrop-filter: blur(var(--odstep-xl));
    padding: var(--uklad-wejscie-wnetrze);
    animation: wejscie var(--czas-krok) var(--krzywa-wejscia) both;
  }
  .ekran:not(.dwie) .karta {
    padding: var(--odstep-7) var(--odstep-7) var(--odstep-sekcja);
  }
  .stopka {
    text-align: center;
    color: var(--kolor-opis);
    font-size: var(--czcionka-drobny);
  }

  @keyframes wejscie {
    from {
      opacity: 0;
      transform: translateY(var(--przesuniecie-karta));
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .marka,
    .karta {
      animation: none;
    }
  }
</style>
