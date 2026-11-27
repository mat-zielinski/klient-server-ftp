<script lang="ts">
  // Okno z jednym polem nazwy: nowy folder albo zmiana nazwy. Sprawdza znaki niedozwolone w Windows.
  import Okno from '../wspolne/Okno.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'

  let {
    otwarte = $bindable(false),
    tytul,
    poczatkowa = '',
    przycisk = 'Zapisz',
    zatwierdz,
  }: {
    otwarte: boolean
    tytul: string
    poczatkowa?: string
    przycisk?: string
    zatwierdz: (nazwa: string) => Promise<boolean>
  } = $props()

  let nazwa = $state('')
  let trwa = $state(false)
  let pole = $state<HTMLInputElement | null>(null)

  $effect(() => {
    if (!otwarte) return
    nazwa = poczatkowa
    // zaznacz nazwę bez rozszerzenia, jak w Eksploratorze
    queueMicrotask(() => {
      if (!pole) return
      pole.focus()
      const kropka = poczatkowa.lastIndexOf('.')
      pole.setSelectionRange(0, kropka > 0 ? kropka : poczatkowa.length)
    })
  })

  const blad = $derived.by(() => {
    const n = nazwa.trim()
    if (!n) return 'Podaj nazwę.'
    if (/[\\/:*?"<>|]/.test(n)) return 'Nazwa nie może zawierać znaków \\ / : * ? " < > |'
    if (n === '.' || n === '..') return 'Niedozwolona nazwa.'
    return ''
  })

  async function zapisz(e?: Event) {
    e?.preventDefault()
    if (blad || trwa) return
    trwa = true
    try {
      if (await zatwierdz(nazwa.trim())) otwarte = false
    } finally {
      trwa = false
    }
  }
</script>

<Okno {tytul} bind:otwarte>
  <form class="tresc" onsubmit={zapisz}>
    <input class="pole" bind:this={pole} bind:value={nazwa} aria-label="Nazwa" spellcheck="false" />
    {#if blad && nazwa}
      <span class="blad">{blad}</span>
    {/if}
  </form>
  {#snippet stopka()}
    <Przycisk rodzaj="duch" onclick={() => (otwarte = false)}>Anuluj</Przycisk>
    <Przycisk rodzaj="glowny" wylaczony={!!blad || trwa} onclick={() => zapisz()}>{przycisk}</Przycisk>
  {/snippet}
</Okno>

<style>
  .tresc {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-s);
  }
  .pole {
    height: var(--wysokosc-pole);
    padding: 0 var(--odstep-m);
    background: var(--kolor-mono);
    border: var(--ramka);
    border-radius: var(--promien-segment);
  }
  .blad {
    color: var(--kolor-czerwony);
    font-size: var(--czcionka-maly);
  }
</style>
