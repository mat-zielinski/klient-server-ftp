<script lang="ts">
  // Ustawienia → Udziały: lista udziałów (nazwa, folder, tylko do odczytu), dodaj / edytuj w oknie, usuń
  // z potwierdzeniem. Zmienia `udzialy` konfiguracji; zapis na serwer przyciskiem [Zapisz] strony.
  import type { Udzial } from '../../api/typy'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Pusto from '../wspolne/Pusto.svelte'
  import Tabela, { type KolumnaTabeli } from '../wspolne/Tabela.svelte'
  import OknoPotwierdzenia from './OknoPotwierdzenia.svelte'
  import OknoUdzialu from './OknoUdzialu.svelte'

  interface Props {
    udzialy: Udzial[]
  }

  let { udzialy = $bindable() }: Props = $props()

  const KOLUMNY: KolumnaTabeli[] = [
    { klucz: 'nazwa', etykieta: 'Nazwa', szerokosc: 'l' },
    { klucz: 'sciezka', etykieta: 'Folder na dysku', ukryjPonizej: 'telefon' },
    { klucz: 'tryb', etykieta: 'Dostęp', szerokosc: 'l' },
    { klucz: 'akcje', etykieta: '', szerokosc: 's', wyrownanie: 'prawo' },
  ]

  let oknoUdzialu = $state(false)
  let oknoUsun = $state(false)
  let edytowany = $state<number>(-1)

  const zajete = $derived(udzialy.filter((_, i) => i !== edytowany).map((u) => u.nazwa))

  function dodaj() {
    edytowany = -1
    oknoUdzialu = true
  }

  function edytuj(i: number) {
    edytowany = i
    oknoUdzialu = true
  }

  function pytajUsun(i: number) {
    edytowany = i
    oknoUsun = true
  }

  function gotowe(u: Udzial) {
    udzialy = edytowany < 0 ? [...udzialy, u] : udzialy.map((s, i) => (i === edytowany ? u : s))
  }

  function usun() {
    udzialy = udzialy.filter((_, i) => i !== edytowany)
  }
</script>

<div class="udzialy">
  <div class="pasek">
    <p class="opis">Foldery na tym komputerze widoczne dla klientów. Prawa kont nadajesz na stronie Użytkownicy.</p>
    <Przycisk rodzaj="glowny" ikona="folder_plus" rozmiar="maly" onclick={dodaj}>Dodaj udział</Przycisk>
  </div>
  <div class="ramka-tabeli">
    <Tabela kolumny={KOLUMNY} wiersze={udzialy} przyklejony={false}>
      {#snippet wiersz(u, i)}
        <tr>
          <td><span class="nazwa">{u.nazwa}</span></td>
          <td class="mono skroc" title={u.sciezka}>{u.sciezka}</td>
          <td>
            <Pigulka stan={u.tylko_odczyt ? 'wyl' : 'ok'}>{u.tylko_odczyt ? 'Tylko odczyt' : 'Odczyt i zapis'}</Pigulka>
          </td>
          <td class="prawo">
            <span class="akcje">
              <Przycisk rodzaj="duch" rozmiar="maly" ikona="olowek" tytul="Edytuj" onclick={() => edytuj(i)} />
              <Przycisk rodzaj="duch" rozmiar="maly" ikona="kosz" tytul="Usuń" onclick={() => pytajUsun(i)} />
            </span>
          </td>
        </tr>
      {/snippet}
      {#snippet pusto()}
        <Pusto ikona="folder" tytul="Brak udziałów" opis="Dodaj pierwszy folder, który zobaczą klienci." />
      {/snippet}
    </Tabela>
  </div>
</div>

<OknoUdzialu bind:otwarte={oknoUdzialu} udzial={edytowany >= 0 ? udzialy[edytowany] : null} {zajete} naGotowe={gotowe} />
<OknoPotwierdzenia tytul="Usunąć udział?" bind:otwarte={oknoUsun} napis="Usuń udział" naPotwierdz={usun}>
  <p>Udział <strong>{udzialy[edytowany]?.nazwa}</strong> zniknie z listy klientów po zapisaniu ustawień.</p>
  <p class="opis">Folder i pliki na dysku zostają nietknięte.</p>
</OknoPotwierdzenia>

<style>
  .udzialy {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-xl);
  }
  .pasek {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--odstep-l);
    flex-wrap: wrap;
  }
  .ramka-tabeli {
    border: var(--ramka);
    border-radius: var(--promien-przycisk);
    overflow: hidden;
  }
  .nazwa {
    font-weight: var(--czcionka-polgruby);
  }
  .akcje {
    display: inline-flex;
    gap: var(--odstep-xs);
  }
</style>
