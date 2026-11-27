<script lang="ts">
  // Okno dodania lub edycji udziału (nazwa, folder na dysku, tylko do odczytu). Zmiana trafia do konfiguracji
  // strony Ustawienia, na serwer dopiero po [Zapisz].
  import type { Udzial } from '../../api/typy'
  import Okno from '../wspolne/Okno.svelte'
  import Pole from '../wspolne/Pole.svelte'
  import Przelacznik from '../wspolne/Przelacznik.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'

  interface Props {
    otwarte: boolean
    /** null = nowy udział. */
    udzial: Udzial | null
    /** Nazwy innych udziałów (sprawdzenie powtórzeń). */
    zajete: string[]
    naGotowe: (u: Udzial) => void
  }

  let { otwarte = $bindable(), udzial, zajete, naGotowe }: Props = $props()

  let nazwa = $state('')
  let sciezka = $state('')
  let tylkoOdczyt = $state(true)
  let pokazBledy = $state(false)

  const bladNazwy = $derived.by(() => {
    const n = nazwa.trim()
    if (!n) return 'Podaj nazwę udziału.'
    if (/[\\/:*?"<>|]/.test(n)) return 'Nazwa nie może zawierać znaków \\ / : * ? " < > |.'
    if (zajete.some((z) => z.toLowerCase() === n.toLowerCase())) return 'Udział o tej nazwie już jest.'
    return ''
  })
  const bladSciezki = $derived(sciezka.trim() ? '' : 'Podaj ścieżkę folderu na dysku.')

  $effect(() => {
    if (!otwarte) return
    nazwa = udzial?.nazwa ?? ''
    sciezka = udzial?.sciezka ?? ''
    tylkoOdczyt = udzial?.tylko_odczyt ?? true
    pokazBledy = false
  })

  function gotowe(e?: Event) {
    e?.preventDefault()
    if (bladNazwy || bladSciezki) {
      pokazBledy = true
      return
    }
    naGotowe({ nazwa: nazwa.trim(), sciezka: sciezka.trim(), tylko_odczyt: tylkoOdczyt })
    otwarte = false
  }
</script>

<Okno tytul={udzial ? `Udział ${udzial.nazwa}` : 'Nowy udział'} bind:otwarte>
  <form class="formularz" onsubmit={gotowe} novalidate>
    <Pole etykieta="Nazwa" bind:value={nazwa} ikona="folder" autofocus
      opis="Tak udział zobaczą klienci (przeglądarka i FTPS)." blad={pokazBledy ? bladNazwy : undefined} />
    <Pole etykieta="Folder na dysku" bind:value={sciezka} ikona="dysk" mono podpowiedz="D:\Udostepnione"
      opis="Pełna ścieżka folderu na tym komputerze." blad={pokazBledy ? bladSciezki : undefined} />
    <Przelacznik etykieta="Tylko do odczytu" bind:value={tylkoOdczyt}
      opis="Blokuje wgrywanie, zmianę nazw i usuwanie dla wszystkich kont." />
    <button type="submit" class="tylko-czytnik" tabindex="-1">Gotowe</button>
  </form>
  {#snippet stopka()}
    <Przycisk rodzaj="duch" onclick={() => (otwarte = false)}>Anuluj</Przycisk>
    <Przycisk rodzaj="glowny" ikona="ok" onclick={() => gotowe()}>{udzial ? 'Zmień' : 'Dodaj'}</Przycisk>
  {/snippet}
</Okno>

<style>
  .formularz {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xl);
  }
</style>
