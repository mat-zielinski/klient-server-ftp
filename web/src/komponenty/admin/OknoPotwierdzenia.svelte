<script lang="ts">
  // Okno potwierdzenia działania (usuń konto, usuń link, usuń udział, reguła zapory): treść + Anuluj / działanie.
  import type { Snippet } from 'svelte'
  import type { NazwaIkony } from '../../ikony/ikony'
  import Okno from '../wspolne/Okno.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'

  interface Props {
    tytul: string
    otwarte: boolean
    /** Napis przycisku działania (np. „Usuń”). */
    napis: string
    ikona?: NazwaIkony
    niebezpieczne?: boolean
    /** Działanie; okno zamyka się, gdy obietnica się spełni (błąd zostawia okno otwarte). */
    naPotwierdz: () => Promise<void> | void
    children: Snippet
  }

  let {
    tytul,
    otwarte = $bindable(),
    napis,
    ikona = 'kosz',
    niebezpieczne = true,
    naPotwierdz,
    children,
  }: Props = $props()

  let trwa = $state(false)

  async function potwierdz() {
    trwa = true
    try {
      await naPotwierdz()
      otwarte = false
    } catch {
      // celowo: komunikat błędu pokazuje wywołujący (powiadom), okno zostaje otwarte
    } finally {
      trwa = false
    }
  }
</script>

<Okno {tytul} bind:otwarte>
  <div class="tresc">{@render children()}</div>
  {#snippet stopka()}
    <Przycisk rodzaj="duch" onclick={() => (otwarte = false)}>Anuluj</Przycisk>
    <Przycisk rodzaj={niebezpieczne ? 'niebezpieczny' : 'glowny'} {ikona} wylaczony={trwa} onclick={potwierdz}>
      {napis}
    </Przycisk>
  {/snippet}
</Okno>

<style>
  .tresc {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-m);
  }
</style>
