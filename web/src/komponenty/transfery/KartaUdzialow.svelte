<script lang="ts">
  // Karta „Zajętość udziałów” (z GET /api/pulpit): pasek zajętości dysku każdego udziału, ostrzeżenie przy
  // małej ilości wolnego miejsca.
  import type { Pulpit } from '../../api/typy'
  import Karta from '../wspolne/Karta.svelte'
  import PasekPostepu from '../wspolne/PasekPostepu.svelte'
  import Pusto from '../wspolne/Pusto.svelte'
  import { procent, rozmiar } from '../../stan/format'
  import { PULPIT } from './stale'

  interface Props { udzialy: Pulpit['udzialy'] }
  let { udzialy }: Props = $props()

  function zajetosc(u: Pulpit['udzialy'][number]): number {
    return u.razem > 0 ? (u.razem - u.wolne) / u.razem : 0
  }
  function stan(u: number): 'ok' | 'ostrz' | 'blad' {
    if (u >= PULPIT.progAlarmu) return 'blad'
    if (u >= PULPIT.progOstrzezenia) return 'ostrz'
    return 'ok'
  }
</script>

<Karta tytul="Zajętość udziałów" ikona="dysk">
  {#if udzialy.length === 0}
    <Pusto ikona="dysk" tytul="Brak udziałów" opis="Dodaj folder w Ustawieniach." />
  {:else}
    <ul>
      {#each udzialy as u (u.nazwa)}
        {@const z = zajetosc(u)}
        <li>
          <div class="wiersz-udzialu">
            <span class="nazwa skroc">{u.nazwa}</span>
            <span class="mono liczby">{procent(z)}</span>
          </div>
          <PasekPostepu wartosc={z} stan={stan(z)} />
          <div class="opis">Wolne {rozmiar(u.wolne)} z {rozmiar(u.razem)}</div>
        </li>
      {/each}
    </ul>
  {/if}
</Karta>

<style>
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--odstep-5);
  }
  li {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-1);
  }
  .wiersz-udzialu {
    display: flex;
    justify-content: space-between;
    gap: var(--odstep-3);
  }
  .nazwa {
    font-weight: var(--czcionka-polgruby);
    min-width: 0;
  }
  .liczby {
    color: var(--kolor-opis);
  }
  .opis {
    font-size: var(--czcionka-maly);
  }
</style>
