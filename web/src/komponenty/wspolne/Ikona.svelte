<script lang="ts">
  // Ikona liniowa 24×24 z src/ikony/ikony.ts; kolor = currentColor (dziedziczy po rodzicu).
  import { IKONY, type NazwaIkony } from '../../ikony/ikony'

  interface Props {
    nazwa: NazwaIkony
    rozmiar?: 's' | 'm' | 'l'
    /** Podpis dla czytnika ekranu; bez niego ikona jest ozdobna (aria-hidden). */
    etykieta?: string
  }

  let { nazwa, rozmiar = 'm', etykieta }: Props = $props()
</script>

<svg
  class="ikona {rozmiar}"
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width="2"
  stroke-linecap="round"
  stroke-linejoin="round"
  role={etykieta ? 'img' : undefined}
  aria-label={etykieta}
  aria-hidden={etykieta ? undefined : 'true'}
>
  {@html IKONY[nazwa] ?? ''}
</svg>

<style>
  .ikona {
    flex: none;
    display: inline-block;
    vertical-align: middle;
    width: var(--uklad-ikona);
    height: var(--uklad-ikona);
  }
  .s {
    width: var(--uklad-ikona-s);
    height: var(--uklad-ikona-s);
  }
  .l {
    width: var(--uklad-ikona-l);
    height: var(--uklad-ikona-l);
  }
</style>
