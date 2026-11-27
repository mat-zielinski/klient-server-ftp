<script lang="ts">
  // Karta (.card stacji): panel z ramką i zaokrągleniem, opcjonalny nagłówek z tytułem, ikoną i akcjami po prawej.
  import type { Snippet } from 'svelte'
  import type { NazwaIkony } from '../../ikony/ikony'
  import Ikona from './Ikona.svelte'

  interface Props {
    tytul?: string
    ikona?: NazwaIkony
    akcje?: Snippet
    children?: Snippet
    /** Treść bez wewnętrznego marginesu (np. tabela do krawędzi karty). */
    bezMarginesu?: boolean
  }

  let { tytul, ikona, akcje, children, bezMarginesu = false }: Props = $props()
</script>

<section class="karta">
  {#if tytul || akcje}
    <header class="naglowek">
      {#if ikona}<span class="ikona"><Ikona nazwa={ikona} /></span>{/if}
      {#if tytul}<h3 class="tytul skroc" title={tytul}>{tytul}</h3>{/if}
      {#if akcje}<div class="akcje">{@render akcje()}</div>{/if}
    </header>
  {/if}
  <div class="tresc" class:bez-marginesu={bezMarginesu}>
    {@render children?.()}
  </div>
</section>

<style>
  .karta {
    display: flex;
    flex-direction: column;
    min-width: 0;
    background: var(--kolor-panel);
    border: var(--ramka);
    border-radius: var(--promien-karta);
  }
  .naglowek {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    min-height: calc(var(--wysokosc-naglowek-karty) + var(--odstep-xl));
    padding: var(--odstep-xl) var(--odstep-xxl) 0;
  }
  .ikona {
    display: inline-flex;
    color: var(--kolor-opis);
  }
  .tytul {
    flex: 1;
    min-width: var(--uklad-tytul-karty-min-w);
    font-size: var(--czcionka-tytul-karty);
    font-weight: var(--czcionka-gruby);
  }
  .akcje {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    margin-left: auto;
  }
  .tresc {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-l);
    padding: var(--odstep-xl) var(--odstep-xxl);
    min-width: 0;
  }
  .bez-marginesu {
    padding: var(--odstep-s) 0 0;
  }
</style>
