<script lang="ts">
  // Paleta poleceń (Ctrl+K): pole wyszukiwania + lista poleceń z rejestru (stan/skroty.svelte.ts), rozmyte
  // dopasowanie z podświetleniem liter, strzałki góra/dół, Enter uruchamia, Esc zamyka.
  import { skroty, type Polecenie, GRUPA_DOMYSLNA } from '../../stan/skroty.svelte'
  import Ikona from '../wspolne/Ikona.svelte'
  import Klawisze from './Klawisze.svelte'
  import { filtruj } from './rozmyte'

  let dialog: HTMLDialogElement | undefined = $state()
  let pole: HTMLInputElement | undefined = $state()
  let lista: HTMLUListElement | undefined = $state()
  let zapytanie = $state('')
  let wybrany = $state(0)

  const wyniki = $derived(
    skroty.paleta
      ? filtruj(skroty.aktywne, zapytanie, (p) => `${p.nazwa} ${p.grupa ?? GRUPA_DOMYSLNA} ${p.slowa ?? ''}`)
      : [],
  )

  $effect(() => {
    if (!dialog) return
    if (skroty.paleta && !dialog.open) {
      zapytanie = ''
      wybrany = 0
      dialog.showModal()
      pole?.focus()
    } else if (!skroty.paleta && dialog.open) dialog.close()
  })

  $effect(() => {
    // nowe zapytanie: zaznaczenie na najlepszym wyniku
    void zapytanie
    wybrany = 0
  })

  $effect(() => {
    lista?.querySelector(`[data-i="${wybrany}"]`)?.scrollIntoView({ block: 'nearest' })
  })

  function uruchom(p: Polecenie) {
    skroty.paleta = false
    // po zamknięciu okna (fokus wraca na stronę), żeby polecenie mogło otworzyć własne okno
    queueMicrotask(() => p.akcja())
  }

  function klawisz(e: KeyboardEvent) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault()
      const n = wyniki.length
      if (n) wybrany = (wybrany + (e.key === 'ArrowDown' ? 1 : -1) + n) % n
    } else if (e.key === 'Enter') {
      e.preventDefault()
      const w = wyniki[wybrany]
      if (w) uruchom(w.e)
    }
  }

  /** Nazwa pocięta na kawałki zwykłe i podświetlone (pozycje dopasowania dotyczą tekstu zaczynającego się od nazwy). */
  function kawalki(nazwa: string, pozycje: number[]): { t: string; p: boolean }[] {
    const zbior = new Set(pozycje.filter((i) => i < nazwa.length))
    const wynik: { t: string; p: boolean }[] = []
    for (let i = 0; i < nazwa.length; i++) {
      const p = zbior.has(i)
      const ost = wynik.at(-1)
      if (ost && ost.p === p) ost.t += nazwa[i]
      else wynik.push({ t: nazwa[i], p })
    }
    return wynik
  }

  function klikZaslony(e: MouseEvent) {
    if (e.target === dialog) skroty.paleta = false
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
<dialog
  bind:this={dialog}
  class="paleta"
  aria-label="Paleta poleceń"
  onclose={() => (skroty.paleta = false)}
  onclick={klikZaslony}
>
  <div class="szukaj">
    <Ikona nazwa="szukaj" />
    <input
      bind:this={pole}
      bind:value={zapytanie}
      type="text"
      placeholder="Wpisz polecenie lub nazwę strony"
      aria-label="Szukaj polecenia"
      role="combobox"
      aria-expanded="true"
      aria-controls="paleta-lista"
      aria-activedescendant={wyniki.length ? `paleta-${wybrany}` : undefined}
      autocomplete="off"
      spellcheck="false"
      onkeydown={klawisz}
    />
    <Klawisze skrot="Esc" />
  </div>
  {#if wyniki.length}
    <ul bind:this={lista} id="paleta-lista" class="lista" role="listbox">
      {#each wyniki as { e: p, d }, i (p.id)}
        <li
          id="paleta-{i}"
          data-i={i}
          role="option"
          aria-selected={i === wybrany}
          class:wybrany={i === wybrany}
          onmousemove={() => (wybrany = i)}
          onclick={() => uruchom(p)}
        >
          <span class="ikona">{#if p.ikona}<Ikona nazwa={p.ikona} rozmiar="s" />{/if}</span>
          <span class="nazwa skroc">
            {#each kawalki(p.nazwa, d.pozycje) as k, j (j)}{#if k.p}<mark>{k.t}</mark>{:else}{k.t}{/if}{/each}
          </span>
          <span class="grupa">{p.grupa ?? GRUPA_DOMYSLNA}</span>
          {#if p.skrot}<Klawisze skrot={p.skrot} />{/if}
        </li>
      {/each}
    </ul>
  {:else}
    <p class="brak">Brak pasujących poleceń.</p>
  {/if}
  <footer class="stopka">
    <span><Klawisze skrot="↑" /> <Klawisze skrot="↓" /> wybór</span>
    <span><Ikona nazwa="enter" rozmiar="s" /> uruchom</span>
    <span><Klawisze skrot="?" /> wszystkie skróty</span>
  </footer>
</dialog>

<style>
  .paleta {
    width: min(var(--uklad-paleta-w), calc(100vw - 2 * var(--odstep-sekcja)));
    margin: var(--uklad-paleta-gora) auto auto;
    padding: 0;
    border: var(--ramka);
    border-radius: var(--promien-karta);
    background: var(--kolor-panel);
    color: var(--kolor-tekst);
    box-shadow: 0 var(--odstep-m) calc(var(--odstep-sekcja) * 2) var(--kolor-cien);
    overflow: hidden;
  }
  .paleta::backdrop {
    background: var(--kolor-zaslona);
  }
  .szukaj {
    display: flex;
    align-items: center;
    gap: var(--odstep-l);
    padding: var(--odstep-xl) var(--odstep-xxl);
    border-bottom: var(--ramka);
    color: var(--kolor-opis);
  }
  input {
    flex: 1;
    min-width: 0;
    border: 0;
    outline: 0;
    background: transparent;
    color: var(--kolor-tekst);
    font-size: var(--czcionka-tytul-karty);
  }
  input::placeholder {
    color: var(--kolor-opis);
  }
  .lista {
    max-height: var(--uklad-paleta-lista-h);
    margin: 0;
    padding: var(--odstep-xs);
    overflow-y: auto;
    list-style: none;
  }
  li {
    display: flex;
    align-items: center;
    gap: var(--odstep-l);
    min-height: var(--wysokosc-pozycja-paska);
    padding: 0 var(--odstep-l);
    border-radius: var(--promien-przycisk);
    cursor: pointer;
  }
  li.wybrany {
    background: var(--kolor-aktywny);
    color: var(--kolor-niebieski);
  }
  .ikona {
    display: inline-flex;
    width: var(--uklad-ikona-s);
    flex: none;
  }
  .nazwa {
    flex: 1;
  }
  mark {
    background: transparent;
    color: var(--kolor-niebieski);
    font-weight: var(--czcionka-gruby);
  }
  .grupa {
    color: var(--kolor-opis);
    font-size: var(--czcionka-drobny);
  }
  .brak {
    padding: var(--odstep-sekcja);
    color: var(--kolor-opis);
    text-align: center;
  }
  .stopka {
    display: flex;
    flex-wrap: wrap;
    gap: var(--odstep-sekcja);
    padding: var(--odstep-s) var(--odstep-xxl);
    border-top: var(--ramka);
    background: var(--kolor-karta);
    color: var(--kolor-opis);
    font-size: var(--czcionka-drobny);
  }
  .stopka span {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-xs);
  }
</style>
