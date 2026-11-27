<script lang="ts">
  // Ruch sieci osobno dla każdego interfejsu (LAN, WireGuard …): nazwa, pigułka rodzaju, niski wykres
  // wysyłania i odbioru. Lista interfejsów z ostatniej próbki; brak = stan pusty.
  import type { Probka } from '../../api/monitor'
  import Karta from '../wspolne/Karta.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import Pusto from '../wspolne/Pusto.svelte'
  import { predkosc } from '../../stan/format'
  import WykresDwuSerii from './WykresDwuSerii.svelte'
  import { seriaPodwojna } from './stan.svelte'
  import { RODZAJE_INTERFEJSU } from './stale'

  interface Props { probki: Probka[]; oknoMs?: number }
  let { probki, oknoMs }: Props = $props()

  const interfejsy = $derived.by(() => {
    const ostatnia = probki[probki.length - 1]
    if (!ostatnia?.siec) return []
    return ostatnia.siec
      .filter((s) => s.rodzaj !== 'petla')
      .map((s) => ({ nazwa: s.interfejs, rodzaj: s.rodzaj }))
      .sort((a, b) => a.rodzaj.localeCompare(b.rodzaj) || a.nazwa.localeCompare(b.nazwa))
  })

  function punkty(nazwa: string) {
    const ruch = (p: Probka) => p.siec?.find((s) => s.interfejs === nazwa)
    return seriaPodwojna(probki, (p) => ruch(p)?.wysylanie_bps ?? 0, (p) => ruch(p)?.odbior_bps ?? 0)
  }
  const opisRodzaju = (r: string) => RODZAJE_INTERFEJSU[r] ?? { nazwa: r || 'Inny', stan: 'wyl' as const }
</script>

<Karta tytul="Sieć wg interfejsu" ikona="siec">
  {#if interfejsy.length === 0}
    <Pusto ikona="siec" tytul="Brak danych o interfejsach" opis="Serwer nie podał jeszcze ruchu poszczególnych interfejsów." />
  {:else}
    <div class="interfejsy">
      {#each interfejsy as i (i.nazwa)}
        <section>
          <header>
            <span class="nazwa" title={i.nazwa}>{i.nazwa}</span>
            <Pigulka stan={opisRodzaju(i.rodzaj).stan}>{opisRodzaju(i.rodzaj).nazwa}</Pigulka>
          </header>
          <WykresDwuSerii punkty={punkty(i.nazwa)} nazwaA="Wysyłanie" nazwaB="Odbiór" barwaA="niebieski" barwaB="zielony"
            formatY={predkosc} {oknoMs} niski />
        </section>
      {/each}
    </div>
  {/if}
</Karta>

<style>
  .interfejsy {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(min(100%, var(--uklad-okno-w)), 1fr));
    gap: var(--odstep-7);
  }
  section {
    min-width: 0;
  }
  header {
    display: flex;
    align-items: center;
    gap: var(--odstep-3);
    margin-bottom: var(--odstep-2);
  }
  .nazwa {
    font-weight: var(--czcionka-polgruby);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
