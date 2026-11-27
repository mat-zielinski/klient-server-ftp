<script lang="ts">
  // Kod QR tekstu (np. linku publicznego do zeskanowania telefonem): własny koder kodqr.ts (bajtowo, korekcja M,
  // wersje 1–10, do 213 bajtów), rysowany w SVG. Zawsze ciemne moduły na jasnym tle (--kolor-qr-*), także w motywie
  // ciemnym, bo tak czytają go aparaty.
  import { zakoduj } from './kodqr'
  import Pusto from './Pusto.svelte'

  interface Props {
    tekst: string
    /** 's' | 'm' | 'l' (tokeny --qr-s/m/l). */
    rozmiar?: 's' | 'm' | 'l'
    etykieta?: string
  }

  let { tekst, rozmiar = 'm', etykieta = 'Kod QR' }: Props = $props()

  const MARGINES = 4 // strefa ciszy wymagana przez normę (moduły)

  const kod = $derived(zakoduj(tekst))
  const bok = $derived(kod ? kod.moduly.length + 2 * MARGINES : 0)
  const sciezka = $derived.by(() => {
    if (!kod) return ''
    const czesci: string[] = []
    kod.moduly.forEach((wiersz, y) =>
      wiersz.forEach((c, x) => {
        if (c) czesci.push(`M${x + MARGINES} ${y + MARGINES}h1v1h-1z`)
      }),
    )
    return czesci.join('')
  })
</script>

{#if kod}
  <svg class="qr {rozmiar}" viewBox="0 0 {bok} {bok}" role="img" aria-label={etykieta} shape-rendering="crispEdges">
    <rect class="tlo" width={bok} height={bok} />
    <path class="moduly" d={sciezka} />
  </svg>
{:else}
  <Pusto ikona="alert" tytul="Tekst za długi na kod QR" opis="Najwyżej 213 bajtów." />
{/if}

<style>
  .qr {
    display: block;
    width: var(--qr-m);
    height: var(--qr-m);
    border-radius: var(--promien-przycisk);
  }
  .s {
    width: var(--qr-s);
    height: var(--qr-s);
  }
  .l {
    width: var(--qr-l);
    height: var(--qr-l);
  }
  .tlo {
    fill: var(--kolor-qr-tlo);
  }
  .moduly {
    fill: var(--kolor-qr-modul);
  }
</style>
