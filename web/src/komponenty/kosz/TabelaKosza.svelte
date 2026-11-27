<script lang="ts">
  // Tabela kosza: nazwa (ikona rodzaju), oryginalne miejsce, kiedy usunięto, rozmiar, akcje Przywróć / Usuń trwale.
  // Na wąskim oknie miejsce i data schodzą pod nazwę (kolumny chowane wg progów z tokenów).
  import type { Snippet } from 'svelte'
  import { IKONA_RODZAJU } from '../../ikony/ikony'
  import Ikona from '../wspolne/Ikona.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Tabela, { type KolumnaTabeli } from '../wspolne/Tabela.svelte'
  import { data, dataWzgledna, rozmiar } from '../../stan/format'
  import { uklad } from '../../stan/uklad.svelte'
  import { folderNadrzedny, sortujKosz, type WpisKosza } from './kosz'

  interface Props {
    wpisy: WpisKosza[]
    /** Wpis, na którym trwa operacja (przyciski wyłączone). */
    zajety?: string
    naPrzywroc: (e: WpisKosza) => void
    naUsun: (e: WpisKosza) => void
    pusto: Snippet
  }

  let { wpisy, zajety = '', naPrzywroc, naUsun, pusto }: Props = $props()

  let sortKlucz = $state('usunieto')
  let malejaco = $state(true)

  const kolumny: KolumnaTabeli[] = [
    { klucz: 'nazwa', etykieta: 'Nazwa' },
    { klucz: 'miejsce', etykieta: 'Oryginalne miejsce', szerokosc: 'xl', ukryjPonizej: 'sredni' },
    { klucz: 'usunieto', etykieta: 'Usunięto', szerokosc: 'l', ukryjPonizej: 'waski' },
    { klucz: 'rozmiar', etykieta: 'Rozmiar', szerokosc: 'm', wyrownanie: 'prawo', ukryjPonizej: 'telefon' },
    { klucz: 'akcje', etykieta: '', szerokosc: 'm', wyrownanie: 'prawo', sortowalna: false },
  ]

  const posortowane = $derived(sortujKosz(wpisy, sortKlucz, malejaco))
  const bezMiejsca = $derived(uklad.ponizej('sredni'))
  const bezDaty = $derived(uklad.ponizej('waski'))
</script>

<Tabela {kolumny} wiersze={posortowane} bind:sortKlucz bind:malejaco naSortuj={() => {}} {pusto}>
  {#snippet wiersz(e)}
    <tr>
      <td>
        <span class="nazwa">
          <span class="ikona" class:folder={e.folder}><Ikona nazwa={IKONA_RODZAJU[e.folder ? 'folder' : e.rodzaj] ?? 'plik'} /></span>
          <span class="teksty">
            <span class="skroc" title={e.sciezka}>{e.nazwa}</span>
            {#if bezMiejsca || bezDaty}
              <span class="opis skroc drugi">
                {#if bezMiejsca}{folderNadrzedny(e.sciezka)}{/if}{#if bezMiejsca && bezDaty} · {/if}{#if bezDaty}{dataWzgledna(e.usunieto)}{/if}
              </span>
            {/if}
          </span>
        </span>
      </td>
      <td><span class="mono opis skroc blok" title={e.sciezka}>{folderNadrzedny(e.sciezka)}</span></td>
      <td><span title={data(e.usunieto)}>{dataWzgledna(e.usunieto)}</span></td>
      <td class="prawo mono">{rozmiar(e.rozmiar)}</td>
      <td class="prawo">
        <span class="akcje">
          <Przycisk rodzaj="duch" rozmiar="maly" ikona="odswiez" tytul="Przywróć do {folderNadrzedny(e.sciezka)}"
            wylaczony={zajety === e.id} onclick={() => naPrzywroc(e)} />
          <Przycisk rodzaj="duch" rozmiar="maly" ikona="kosz" tytul="Usuń trwale" wylaczony={zajety === e.id}
            onclick={() => naUsun(e)} />
        </span>
      </td>
    </tr>
  {/snippet}
</Tabela>

<style>
  .nazwa {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    min-width: 0;
  }
  .ikona {
    display: inline-flex;
    flex: none;
    color: var(--kolor-opis);
  }
  .ikona.folder {
    color: var(--kolor-zolty);
  }
  .teksty {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .blok {
    display: block;
  }
  .akcje {
    display: inline-flex;
    gap: var(--odstep-xs);
  }
</style>
