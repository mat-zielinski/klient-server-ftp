<script lang="ts">
  // Kolejka wgrywania pod listą plików: postęp każdego pliku, anuluj, ponów, nadpisz istniejący.
  import Karta from '../wspolne/Karta.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import PasekPostepu from '../wspolne/PasekPostepu.svelte'
  import { predkosc, rozmiar } from '../../stan/format'
  import type { KolejkaWgrywania, StanWgrywania } from './wgrywanie.svelte'

  let { kolejka }: { kolejka: KolejkaWgrywania } = $props()

  const OPISY: Record<StanWgrywania, { stan: 'ok' | 'info' | 'blad' | 'ostrz' | 'wyl'; tekst: string }> = {
    czeka: { stan: 'wyl', tekst: 'Czeka' },
    trwa: { stan: 'info', tekst: 'Wgrywanie' },
    gotowe: { stan: 'ok', tekst: 'Gotowe' },
    blad: { stan: 'blad', tekst: 'Błąd' },
    istnieje: { stan: 'ostrz', tekst: 'Istnieje' },
    anulowane: { stan: 'wyl', tekst: 'Anulowano' },
  }
</script>

{#if kolejka.zadania.length}
  <Karta tytul="Wgrywanie ({kolejka.aktywne})" ikona="wgraj">
    {#snippet akcje()}
      <Przycisk rodzaj="duch" rozmiar="maly" onclick={() => kolejka.wyczyscZakonczone()}>Wyczyść zakończone</Przycisk>
    {/snippet}
    <ul class="lista">
      {#each kolejka.zadania as z (z.id)}
        {@const u = z.plik.size ? z.przeslano / z.plik.size : z.stan === 'gotowe' ? 1 : 0}
        <li class="pozycja">
          <span class="nazwa skroc" title={z.sciezka}>{z.sciezka}</span>
          <Pigulka stan={OPISY[z.stan].stan}>{OPISY[z.stan].tekst}</Pigulka>
          <div class="pasek">
            <PasekPostepu
              wartosc={z.stan === 'czeka' ? 0 : u}
              stan={z.stan === 'blad' ? 'blad' : z.stan === 'gotowe' ? 'ok' : 'info'}
              etykieta="Postęp wgrywania"
            />
          </div>
          <span class="opis dane">
            {#if z.komunikat}
              {z.komunikat}
            {:else}
              {rozmiar(z.przeslano)} z {rozmiar(z.plik.size)}{z.stan === 'trwa' ? `, ${predkosc(z.predkosc)}` : ''}
            {/if}
          </span>
          <span class="akcje">
            {#if z.stan === 'istnieje'}
              <Przycisk rodzaj="niebezpieczny" rozmiar="maly" onclick={() => kolejka.nadpisz(z.id)}>Nadpisz</Przycisk>
              <Przycisk rodzaj="duch" rozmiar="maly" onclick={() => kolejka.usun(z.id)}>Pomiń</Przycisk>
            {:else if z.stan === 'blad'}
              <Przycisk rozmiar="maly" ikona="odswiez" onclick={() => kolejka.ponow(z.id)}>Wznów</Przycisk>
              <Przycisk rodzaj="duch" rozmiar="maly" ikona="zamknij" tytul="Usuń z listy" onclick={() => kolejka.usun(z.id)} />
            {:else if z.stan === 'trwa' || z.stan === 'czeka'}
              <Przycisk rodzaj="duch" rozmiar="maly" ikona="zamknij" tytul="Anuluj" onclick={() => kolejka.anuluj(z.id)} />
            {/if}
          </span>
        </li>
      {/each}
    </ul>
  </Karta>
{/if}

<style>
  .lista {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-s);
    max-height: calc(var(--wysokosc-wiersz-tabeli) * 6);
    overflow-y: auto;
  }
  .pozycja {
    display: grid;
    grid-template-columns: minmax(0, 2fr) auto minmax(var(--kolumna-m), 1fr) minmax(0, 1.2fr) auto;
    align-items: center;
    gap: var(--odstep-m);
    min-height: var(--wysokosc-wiersz-tabeli);
  }
  .nazwa {
    font-weight: var(--czcionka-polgruby);
  }
  .dane {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .akcje {
    display: flex;
    gap: var(--odstep-xs);
  }
</style>
