<script lang="ts" module>
  // Ocena siły i generator haseł (czyste funkcje, bez stanu): używa ich PoleHasla, można importować osobno.

  /** Poziom siły: 0 = brak hasła, 1 Słabe, 2 Średnie, 3 Dobre, 4 Silne. */
  export type SilaHasla = 0 | 1 | 2 | 3 | 4

  export const OPISY_SILY: Record<SilaHasla, string> = { 0: '', 1: 'Słabe', 2: 'Średnie', 3: 'Dobre', 4: 'Silne' }

  /** Liczba segmentów paska siły. */
  export const SEGMENTY_SILY = 4
  /** Długości dające kolejne punkty (ponad minimum). */
  const DLUGOSC_DOBRA = 14
  const DLUGOSC_SILNA = 20
  /** Ile razy z rzędu ten sam znak to już „powtórzenie”. */
  const POWTORZENIE = 3
  /** Długość generowanego hasła. */
  export const DLUGOSC_GENEROWANEGO = 16
  /** Alfabet generatora bez znaków mylących (0 O o 1 l I |), bez cudzysłowów i spacji. */
  const KLASY = ['abcdefghijkmnpqrstuvwxyz', 'ABCDEFGHJKLMNPQRSTUVWXYZ', '23456789', '!@$%&*-_=+?']

  function klasyZnakow(h: string): number {
    return [/[a-z]/, /[A-Z]/, /\d/, /[^a-zA-Z\d]/].filter((r) => r.test(h)).length
  }

  function maPowtorzenia(h: string): boolean {
    let ciag = 1
    for (let i = 1; i < h.length; i++) {
      ciag = h[i] === h[i - 1] ? ciag + 1 : 1
      if (ciag >= POWTORZENIE) return true
    }
    return new Set(h).size * 2 < h.length
  }

  /** Prosta ocena siły: długość względem minimum, klasy znaków, kara za powtórzenia. */
  export function ocenHaslo(h: string, minDlugosc: number): SilaHasla {
    if (!h) return 0
    if (h.length < minDlugosc) return 1
    let punkty = 1
    if (h.length >= DLUGOSC_DOBRA) punkty++
    if (h.length >= DLUGOSC_SILNA) punkty++
    punkty += Math.max(0, klasyZnakow(h) - 1)
    if (maPowtorzenia(h)) punkty -= 2
    if (punkty >= 5) return 4
    if (punkty >= 4) return 3
    return punkty >= 2 ? 2 : 1
  }

  /** Losowa liczba 0..n-1 bez przekrzywienia (odrzucanie wartości z niepełnego zakresu). */
  function losowa(n: number): number {
    const zakres = Math.floor(0x100000000 / n) * n
    const bufor = new Uint32Array(1)
    do crypto.getRandomValues(bufor)
    while (bufor[0] >= zakres)
    return bufor[0] % n
  }

  /** Hasło z crypto.getRandomValues: każda klasa znaków co najmniej raz, bez powtórzeń z rzędu. */
  export function generujHaslo(dlugosc = DLUGOSC_GENEROWANEGO): string {
    const wszystkie = KLASY.join('')
    for (;;) {
      let h = ''
      for (let i = 0; i < dlugosc; i++) h += wszystkie[losowa(wszystkie.length)]
      if (klasyZnakow(h) === KLASY.length && !maPowtorzenia(h)) return h
    }
  }
</script>

<script lang="ts">
  // Pole hasła: przycisk oka (pokaż / ukryj, nie zabiera fokusu), ostrzeżenie o Caps Lock, opcjonalnie pasek siły
  // (4 segmenty + opis) i przycisk „Wygeneruj” (hasło 16 znaków, od razu widoczne, z przyciskiem Kopiuj).
  import type { NazwaIkony } from '../../ikony/ikony'
  import { powiadom } from '../../stan/powiadomienia.svelte'
  import Ikona from './Ikona.svelte'

  interface Props {
    etykieta: string
    value: string
    autocomplete?: HTMLInputElement['autocomplete']
    ikona?: NazwaIkony
    podpowiedz?: string
    blad?: string
    opis?: string
    autofocus?: boolean
    /** Hasło widoczne jawnym tekstem (bindable: kreator pokazuje oba pola po wygenerowaniu). */
    pokaz?: boolean
    /** Pasek siły hasła pod polem. */
    sila?: boolean
    /** Minimalna długość (dla oceny siły). */
    minDlugosc?: number
    /** Przycisk „Wygeneruj” nad polem. */
    generator?: boolean
    /** Wołane po wygenerowaniu (np. wstawienie hasła także do pola powtórzenia). */
    onwygeneruj?: (haslo: string) => void
    /** Zielony znaczek w polu (np. powtórzenie zgodne z hasłem). */
    zgodne?: boolean
    onblur?: (e: FocusEvent) => void
    onkeydown?: (e: KeyboardEvent) => void
  }

  let {
    etykieta,
    value = $bindable(),
    autocomplete = 'current-password',
    ikona = 'klodka',
    podpowiedz,
    blad,
    opis,
    autofocus = false,
    pokaz = $bindable(false),
    sila = false,
    minDlugosc = 1,
    generator = false,
    onwygeneruj,
    zgodne = false,
    onblur,
    onkeydown,
  }: Props = $props()

  const id = $props.id()
  let capsLock = $state(false)
  let wygenerowane = $state('')

  const ocena = $derived(sila ? ocenHaslo(value, minDlugosc) : 0)
  const pokazKopiuj = $derived(!!wygenerowane && wygenerowane === value)

  function ustawFokus(el: HTMLInputElement) {
    if (autofocus) el.focus()
  }

  function sprawdzCaps(e: KeyboardEvent) {
    // getModifierState bywa niedostępne w zdarzeniach syntetycznych (autouzupełnianie): wtedy bez zmiany
    if (typeof e.getModifierState === 'function') capsLock = e.getModifierState('CapsLock')
  }

  function klawisz(e: KeyboardEvent) {
    sprawdzCaps(e)
    onkeydown?.(e)
  }

  function opuszczone(e: FocusEvent) {
    capsLock = false
    onblur?.(e)
  }

  function wygeneruj() {
    const h = generujHaslo()
    value = h
    wygenerowane = h
    pokaz = true
    onwygeneruj?.(h)
  }

  async function kopiuj() {
    try {
      await navigator.clipboard.writeText(value)
      powiadom('Hasło skopiowane do schowka.', 'ok')
    } catch {
      powiadom('Nie udało się skopiować. Zaznacz hasło i skopiuj ręcznie.', 'blad')
    }
  }
</script>

<div class="pole" class:z-bledem={!!blad}>
  <div class="naglowek">
    <label class="etykieta-pola" for={id}>{etykieta}</label>
    {#if generator}
      <button type="button" class="link" onclick={wygeneruj}>
        <Ikona nazwa="generuj" rozmiar="s" />Wygeneruj
      </button>
    {/if}
  </div>
  <span class="ramka">
    <span class="ikona"><Ikona nazwa={ikona} rozmiar="s" /></span>
    <input
      {id}
      bind:value
      type={pokaz ? 'text' : 'password'}
      class:jawne={pokaz}
      placeholder={podpowiedz}
      {autocomplete}
      autocapitalize="off"
      spellcheck="false"
      aria-invalid={blad ? 'true' : undefined}
      aria-describedby="{id}-opis"
      onkeydown={klawisz}
      onkeyup={sprawdzCaps}
      onblur={opuszczone}
      use:ustawFokus
    />
    {#if zgodne}
      <span class="zgodne" title="Hasła są takie same">
        <Ikona nazwa="ok_kolo" rozmiar="s" etykieta="Hasła są takie same" />
      </span>
    {/if}
    {#if pokazKopiuj}
      <button type="button" class="oko" title="Kopiuj hasło" aria-label="Kopiuj hasło"
        onmousedown={(e) => e.preventDefault()} onclick={kopiuj}>
        <Ikona nazwa="kopiuj" rozmiar="s" />
      </button>
    {/if}
    <button
      type="button"
      class="oko"
      title={pokaz ? 'Ukryj hasło' : 'Pokaż hasło'}
      aria-label={pokaz ? 'Ukryj hasło' : 'Pokaż hasło'}
      aria-pressed={pokaz}
      aria-controls={id}
      onmousedown={(e) => e.preventDefault()}
      onclick={() => (pokaz = !pokaz)}
    >
      <Ikona nazwa={pokaz ? 'oko_zamkniete' : 'oko'} rozmiar="s" />
    </button>
  </span>

  {#if sila && value}
    <div class="sila poziom-{ocena}" aria-live="polite">
      <span class="segmenty" aria-hidden="true">
        {#each { length: SEGMENTY_SILY } as _, i (i)}<span class="segment" class:pelny={i < ocena}></span>{/each}
      </span>
      <span class="opis-sily">Siła hasła: <strong>{OPISY_SILY[ocena]}</strong></span>
    </div>
  {/if}

  <div id="{id}-opis" class="pod">
    {#if capsLock}
      <span class="caps" role="status"><Ikona nazwa="caps_lock" rozmiar="s" />Włączony Caps Lock</span>
    {/if}
    {#if blad}
      <span class="blad">{blad}</span>
    {:else if pokazKopiuj}
      <span class="opis">Wygenerowane hasło jest widoczne. Skopiuj je i zapisz w menedżerze haseł.</span>
    {:else if opis}
      <span class="opis">{opis}</span>
    {/if}
  </div>
</div>

<style>
  .pole {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xs);
    min-width: 0;
  }
  .naglowek {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--odstep-m);
  }
  .etykieta-pola {
    font-size: var(--czcionka-maly);
    font-weight: var(--czcionka-polgruby);
    color: var(--kolor-opis);
  }
  .link {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-xs);
    padding: 0;
    border: 0;
    background: none;
    color: var(--kolor-niebieski);
    font-size: var(--czcionka-maly);
    font-weight: var(--czcionka-polgruby);
    cursor: pointer;
  }
  .link:hover {
    text-decoration: underline;
  }
  .ramka {
    display: flex;
    align-items: center;
    gap: var(--odstep-s);
    height: var(--wysokosc-pole-duze);
    padding: 0 var(--odstep-xs) 0 var(--odstep-przycisk-maly);
    background: var(--kolor-karta);
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
    transition: border-color var(--czas-szybki), box-shadow var(--czas-szybki);
  }
  .ramka:focus-within {
    border-color: var(--kolor-niebieski);
    box-shadow: 0 0 0 var(--odstep-xs) var(--kolor-fokus-pola);
  }
  .z-bledem .ramka {
    border-color: var(--kolor-czerwony);
  }
  .ikona {
    display: inline-flex;
    color: var(--kolor-opis);
  }
  input {
    flex: 1;
    min-width: 0;
    height: 100%;
    padding: 0;
    border: 0;
    outline: 0;
    background: transparent;
    font-size: var(--czcionka-tekst);
  }
  input.jawne {
    font-family: var(--czcionka-mono);
  }
  input::placeholder {
    color: var(--kolor-opis);
    font-family: var(--czcionka-ui);
  }
  .zgodne {
    display: inline-flex;
    color: var(--kolor-zielony);
  }
  .oko {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--wysokosc-przycisk);
    height: var(--wysokosc-przycisk);
    padding: 0;
    border: 0;
    border-radius: var(--promien-segment);
    background: transparent;
    color: var(--kolor-opis);
    cursor: pointer;
  }
  .oko:hover {
    background: var(--kolor-przycisk);
    color: var(--kolor-tekst);
  }
  .oko:focus-visible {
    outline: var(--odstep-ramka) solid var(--kolor-niebieski);
  }
  .sila {
    display: flex;
    align-items: center;
    gap: var(--odstep-m);
    font-size: var(--czcionka-drobny);
    color: var(--kolor-opis);
  }
  .segmenty {
    flex: 1;
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--odstep-xs);
  }
  .segment {
    height: var(--wysokosc-sila-hasla);
    border-radius: var(--promien-kropka);
    background: var(--kolor-siatka);
    transition: background var(--czas-sredni);
  }
  .poziom-1 .pelny {
    background: var(--kolor-czerwony);
  }
  .poziom-2 .pelny {
    background: var(--kolor-pomaranczowy);
  }
  .poziom-3 .pelny {
    background: var(--kolor-zolty);
  }
  .poziom-4 .pelny {
    background: var(--kolor-zielony);
  }
  .poziom-1 strong {
    color: var(--kolor-czerwony);
  }
  .poziom-2 strong {
    color: var(--kolor-pomaranczowy);
  }
  .poziom-3 strong {
    color: var(--kolor-zolty);
  }
  .poziom-4 strong {
    color: var(--kolor-zielony);
  }
  .opis-sily {
    flex: none;
    text-align: right;
  }
  .pod {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xs);
    font-size: var(--czcionka-drobny);
  }
  .pod:empty {
    display: none;
  }
  .caps {
    display: inline-flex;
    align-items: center;
    gap: var(--odstep-xs);
    color: var(--kolor-zolty);
    font-weight: var(--czcionka-polgruby);
  }
  .blad {
    color: var(--kolor-czerwony);
  }
</style>
