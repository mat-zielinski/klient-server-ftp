<script lang="ts">
  // Ekran bez paska bocznego (Logowanie, Kreator, link publiczny) w stylu UniFi: granatowe tło, logo nad
  // wyśrodkowaną kartą, wersja w stopce. Wąskie okno: karta na całą szerokość z marginesem.
  import type { Snippet } from 'svelte'
  import Ikona from '../wspolne/Ikona.svelte'
  import { sesja } from '../../stan/sesja.svelte'

  interface Props {
    /** Podtytuł pod logo (np. „Zaloguj się, aby kontynuować”). */
    podtytul?: string
    /** Szersza karta (kreator, lista plików folderu). */
    szeroka?: boolean
    children: Snippet
  }

  let { podtytul, szeroka = false, children }: Props = $props()

  const nazwa = $derived(sesja.stan?.nazwa || 'Skrytka')
  const wersja = $derived(sesja.stan?.wersja ?? '')
</script>

<div class="ekran">
  <div class="srodek" class:szeroka>
    <div class="logo">
      <span class="znak"><Ikona nazwa="serwer" rozmiar="l" /></span>
      <span class="nazwa">{nazwa}</span>
    </div>
    {#if podtytul}<p class="podtytul">{podtytul}</p>{/if}
    <div class="karta">
      {@render children()}
    </div>
    {#if wersja}<p class="wersja">Wersja {wersja}</p>{/if}
  </div>
</div>

<style>
  .ekran {
    min-height: 100%;
    border-radius: var(--promien-karta);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--odstep-7);
    background:
      radial-gradient(ellipse at top, var(--kolor-tlo-info), transparent 60%),
      var(--kolor-tlo);
  }
  .srodek {
    width: 100%;
    max-width: var(--uklad-okno-w);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--odstep-l);
    margin: auto;
  }
  .srodek:not(.szeroka) {
    max-width: calc(var(--uklad-okno-w) * 0.8);
  }
  .srodek.szeroka {
    max-width: calc(var(--uklad-okno-w) * 1.5);
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
    background: var(--kolor-niebieski-tlo);
    color: var(--kolor-na-ciemnym);
  }
  .nazwa {
    font-size: var(--czcionka-duzy);
    font-weight: var(--czcionka-gruby);
  }
  .podtytul {
    color: var(--kolor-opis);
    text-align: center;
  }
  .karta {
    width: 100%;
    background: var(--kolor-panel);
    border: var(--ramka);
    border-radius: var(--promien-karta);
    box-shadow: 0 var(--odstep-m) var(--odstep-sekcja) var(--kolor-cien);
    padding: var(--odstep-7);
  }
  .wersja {
    color: var(--kolor-opis);
    font-size: var(--czcionka-drobny);
  }
</style>
