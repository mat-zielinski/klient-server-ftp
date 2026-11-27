<script lang="ts">
  // Ustawienia → Wygląd: motyw ciemny / jasny. Wybór od razu zmienia wygląd w tej przeglądarce, a po [Zapisz]
  // staje się motywem domyślnym serwera dla wszystkich, którzy nie wybrali własnego.
  import type { Konfiguracja } from '../../api/typy'
  import type { NazwaIkony } from '../../ikony/ikony'
  import Ikona from '../wspolne/Ikona.svelte'
  import { motyw, type Motyw } from '../../stan/motyw.svelte'

  interface Props {
    konfig: Konfiguracja
  }

  let { konfig = $bindable() }: Props = $props()

  const MOTYWY: { id: Motyw; nazwa: string; opis: string; ikona: NazwaIkony }[] = [
    { id: 'ciemny', nazwa: 'Ciemny', opis: 'Granatowe tło jak w stacji. Domyślny.', ikona: 'ksiezyc' },
    { id: 'jasny', nazwa: 'Jasny', opis: 'Jasne tło, lepszy w mocnym świetle.', ikona: 'slonce' },
  ]

  const wybrany = $derived<Motyw>(konfig.motyw === 'jasny' ? 'jasny' : 'ciemny')

  function wybierz(m: Motyw) {
    konfig.motyw = m
    motyw.ustaw(m)
  }
</script>

<div class="wyglad">
  <h4>Motyw</h4>
  <div class="motywy" role="radiogroup" aria-label="Motyw">
    {#each MOTYWY as m (m.id)}
      <button type="button" role="radio" aria-checked={wybrany === m.id} class="motyw" class:wybrany={wybrany === m.id}
        onclick={() => wybierz(m.id)}>
        <span class="tytul"><Ikona nazwa={m.ikona} rozmiar="s" />{m.nazwa}</span>
        <span class="opis">{m.opis}</span>
      </button>
    {/each}
  </div>
  <p class="opis">
    Każdy użytkownik może też przełączyć motyw u siebie; jego wybór zapamięta przeglądarka.
  </p>
</div>

<style>
  .wyglad {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xl);
  }
  .motywy {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(var(--kolumna-xl), 1fr));
    gap: var(--odstep-l);
  }
  .motyw {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--odstep-s);
    padding: var(--odstep-l);
    text-align: left;
    border: var(--ramka);
    border-radius: var(--promien-kafel);
    background: var(--kolor-karta);
  }
  .motyw:hover {
    background: var(--kolor-przycisk);
  }
  .motyw.wybrany {
    border-color: var(--kolor-niebieski);
    background: var(--kolor-aktywny);
  }
  .tytul {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-s);
    font-weight: var(--czcionka-polgruby);
  }
</style>
