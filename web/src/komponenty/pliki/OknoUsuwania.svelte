<script lang="ts">
  // Potwierdzenie usunięcia (do kosza serwera); usuwa po kolei i zgłasza każdy błąd osobno.
  import type { Wpis } from '../../api/typy'
  import Okno from '../wspolne/Okno.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import { powiadom } from '../../stan/powiadomienia.svelte'
  import { usun } from './operacje'

  let {
    otwarte = $bindable(false),
    udzial,
    wpisy,
    poUsunieciu,
  }: { otwarte: boolean; udzial: string; wpisy: Wpis[]; poUsunieciu: () => void } = $props()

  let trwa = $state(false)

  async function usunTeraz() {
    trwa = true
    const bledy = await usun(udzial, wpisy)
    trwa = false
    otwarte = false
    const ok = wpisy.length - bledy.length
    if (ok) powiadom(ok === 1 ? 'Przeniesiono do kosza.' : `Przeniesiono do kosza: ${ok}.`, 'ok')
    for (const b of bledy) powiadom(`${b.wpis.nazwa}: ${b.blad}`, 'blad')
    poUsunieciu()
  }
</script>

<Okno tytul="Przenieść do kosza?" bind:otwarte>
  {#if wpisy.length === 1}
    <p>Element <b>{wpisy[0].nazwa}</b> trafi do kosza serwera.</p>
  {:else}
    <p>Zaznaczone elementy ({wpisy.length}) trafią do kosza serwera.</p>
  {/if}
  {#snippet stopka()}
    <Przycisk rodzaj="duch" onclick={() => (otwarte = false)}>Anuluj</Przycisk>
    <Przycisk rodzaj="niebezpieczny" ikona="kosz" wylaczony={trwa} onclick={usunTeraz}>Usuń</Przycisk>
  {/snippet}
</Okno>
