<script lang="ts" module>
  // Licznik wystąpień: każde logo dostaje własne identyfikatory gradientów (przy wspólnych id gradient z ukrytej
  // kopii nie rysuje się w widocznej).
  let licznik = 0

  // CSP strony (style-src 'self') blokuje atrybuty style w SVG wstawionym przez {@html}, więc kolory z tokenów
  // przechodzą z atrybutów style na klasy, a klasy kolorują reguły poniżej.
  const STYLE_NA_KLASY: [string, string][] = [
    ['style="stop-color:var(--kolor-niebieski)"', 'class="logo-jasny"'],
    ['style="stop-color:var(--kolor-niebieski-tlo)"', 'class="logo-ciemny"'],
    ['style="stop-color:var(--kolor-cien);stop-opacity:0"', 'class="logo-cien-0"'],
    ['style="stop-color:var(--kolor-cien)"', 'class="logo-cien"'],
    ['style="fill:var(--kolor-na-ciemnym)"', 'class="logo-znak"'],
  ]
</script>

<script lang="ts">
  import { LOGO_SVG } from '../../ikony/logo'

  const svg = STYLE_NA_KLASY.reduce(
    (s, [styl, klasa]) => s.replaceAll(styl, klasa),
    LOGO_SVG.replaceAll('skrytka-logo-', `skrytka-logo-${++licznik}-`),
  )
</script>

<span class="logo-aplikacji">{@html svg}</span>

<style>
  .logo-aplikacji {
    display: contents;
  }
  .logo-aplikacji :global(.logo-jasny) {
    stop-color: var(--kolor-niebieski);
  }
  .logo-aplikacji :global(.logo-ciemny) {
    stop-color: var(--kolor-niebieski-tlo);
  }
  .logo-aplikacji :global(.logo-cien) {
    stop-color: var(--kolor-cien);
  }
  .logo-aplikacji :global(.logo-cien-0) {
    stop-color: var(--kolor-cien);
    stop-opacity: 0;
  }
  .logo-aplikacji :global(.logo-znak) {
    fill: var(--kolor-na-ciemnym);
  }
</style>
