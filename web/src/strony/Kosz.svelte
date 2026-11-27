<script lang="ts">
  // Kosz: usunięte pliki i foldery wybranego udziału (tylko udziały z prawem zapisu). Przywróć (zajęte miejsce =
  // pytanie o nadpisanie; obecny element trafia wtedy do kosza), Usuń trwale (potwierdzenie, nieodwracalne),
  // Opróżnij kosz (tylko admin). Serwer sam usuwa wpisy starsze niż DNI_AUTOMATYCZNEGO_OPROZNIANIA.
  import { onMount } from 'svelte'
  import { api, BladApi, trescBledu } from '../api/klient'
  import type { UdzialInfo } from '../api/typy'
  import OknoPotwierdzenia from '../komponenty/admin/OknoPotwierdzenia.svelte'
  import TabelaKosza from '../komponenty/kosz/TabelaKosza.svelte'
  import {
    DNI_AUTOMATYCZNEGO_OPROZNIANIA,
    folderNadrzedny,
    listaKosza,
    oproznij,
    przywroc,
    udzialyZKoszem,
    usunTrwale,
    type WpisKosza,
  } from '../komponenty/kosz/kosz'
  import NaglowekStrony from '../komponenty/NaglowekStrony.svelte'
  import Karta from '../komponenty/wspolne/Karta.svelte'
  import Przycisk from '../komponenty/wspolne/Przycisk.svelte'
  import Pusto from '../komponenty/wspolne/Pusto.svelte'
  import Szkielet from '../komponenty/wspolne/Szkielet.svelte'
  import Wybor from '../komponenty/wspolne/Wybor.svelte'
  import { rozmiar } from '../stan/format'
  import { idz } from '../stan/nawigacja.svelte'
  import { powiadom } from '../stan/powiadomienia.svelte'
  import { sesja } from '../stan/sesja.svelte'

  let udzialy = $state<UdzialInfo[]>([])
  let udzialyWczytane = $state(false)
  let bladUdzialow = $state('')
  let udzial = $state('')
  let wpisy = $state<WpisKosza[]>([])
  let wczytano = $state(false)
  let blad = $state('')
  let zajety = $state('')
  let wybrany = $state<WpisKosza | null>(null)
  let oknoNadpisz = $state(false)
  let oknoUsun = $state(false)
  let oknoOproznij = $state(false)

  const suma = $derived(wpisy.reduce((s, e) => s + e.rozmiar, 0))

  async function wczytajUdzialy() {
    try {
      udzialy = udzialyZKoszem(await api.get<UdzialInfo[]>('/udzialy'))
      bladUdzialow = ''
      if (!udzialy.some((u) => u.nazwa === udzial)) udzial = udzialy[0]?.nazwa ?? ''
      if (udzial) await wczytaj()
    } catch (b) {
      bladUdzialow = trescBledu(b)
    } finally {
      udzialyWczytane = true
    }
  }

  async function wczytaj() {
    if (!udzial) return
    const dla = udzial
    try {
      const l = await listaKosza(dla)
      if (dla !== udzial) return
      wpisy = l
      blad = ''
    } catch (b) {
      if (dla === udzial) blad = trescBledu(b)
    } finally {
      if (dla === udzial) wczytano = true
    }
  }

  function zmienUdzial(nazwa: string) {
    udzial = nazwa
    wpisy = []
    wczytano = false
    blad = ''
    void wczytaj()
  }

  async function przywrocWpis(e: WpisKosza, nadpisz: boolean) {
    zajety = e.id
    try {
      await przywroc(udzial, e.id, nadpisz)
      powiadom(`Przywrócono „${e.nazwa}” do ${folderNadrzedny(e.sciezka)}.`, 'ok')
      await wczytaj()
    } catch (b) {
      if (!nadpisz && b instanceof BladApi && b.kod === 409) {
        wybrany = e
        oknoNadpisz = true
        return
      }
      powiadom(trescBledu(b), 'blad')
      if (nadpisz) throw b
    } finally {
      zajety = ''
    }
  }

  async function przywrocZNadpisaniem() {
    if (wybrany) await przywrocWpis(wybrany, true)
  }

  function pytajUsun(e: WpisKosza) {
    wybrany = e
    oknoUsun = true
  }

  async function usunWybrany() {
    const e = wybrany
    if (!e) return
    try {
      await usunTrwale(udzial, e.id)
      powiadom(`Usunięto trwale „${e.nazwa}”.`, 'ok')
      await wczytaj()
    } catch (b) {
      powiadom(trescBledu(b), 'blad')
      throw b
    }
  }

  async function oproznijKosz() {
    try {
      const w = await oproznij(udzial)
      powiadom(`Opróżniono kosz udziału ${udzial} (usunięto ${w.usuniete}).`, 'ok')
      await wczytaj()
    } catch (b) {
      powiadom(trescBledu(b), 'blad')
      throw b
    }
  }

  onMount(() => {
    void wczytajUdzialy()
  })
</script>

<div class="strona">
  <NaglowekStrony tytul="Kosz" ikona="kosz" podtytul="Usunięte pliki: przywracanie i trwałe usuwanie">
    {#snippet akcje()}
      {#if udzialy.length > 1}
        <div class="wybor-udzialu">
          <Wybor etykieta="Udział" ukryjEtykiete value={udzial} opcje={udzialy.map((u) => u.nazwa)} onchange={zmienUdzial} />
        </div>
      {/if}
    {/snippet}
  </NaglowekStrony>

  {#if !udzialyWczytane}
    <Karta><Szkielet /></Karta>
  {:else if bladUdzialow}
    <Karta>
      <Pusto ikona="blad" tytul="Nie udało się wczytać udziałów" opis={bladUdzialow}>
        <Przycisk ikona="odswiez" onclick={wczytajUdzialy}>Spróbuj ponownie</Przycisk>
      </Pusto>
    </Karta>
  {:else if udzialy.length === 0}
    <Karta>
      <Pusto ikona="kosz" tytul="Brak koszy do przeglądania"
        opis="Kosz ma każdy udział, w którym możesz usuwać pliki. Nie masz prawa zapisu w żadnym udziale." />
    </Karta>
  {:else}
    <Karta tytul="Kosz udziału {udzial}" ikona="kosz" bezMarginesu>
      {#snippet akcje()}
        {#if wpisy.length}<span class="licznik">{wpisy.length} · {rozmiar(suma)}</span>{/if}
        <Przycisk rodzaj="duch" ikona="odswiez" rozmiar="maly" tytul="Odśwież" onclick={wczytaj} />
        {#if sesja.admin}
          <Przycisk rodzaj="niebezpieczny" ikona="kosz" rozmiar="maly" wylaczony={!wpisy.length}
            tytul={wpisy.length ? 'Usuń trwale wszystko z kosza tego udziału' : 'Kosz jest pusty'}
            onclick={() => (oknoOproznij = true)}>Opróżnij kosz</Przycisk>
        {/if}
      {/snippet}
      {#if blad}
        <Pusto ikona="blad" tytul="Nie udało się wczytać kosza" opis={blad}>
          <Przycisk ikona="odswiez" onclick={wczytaj}>Spróbuj ponownie</Przycisk>
        </Pusto>
      {:else if !wczytano}
        <div class="szkielet"><Szkielet /></div>
      {:else}
        <TabelaKosza {wpisy} {zajety} naPrzywroc={(e) => przywrocWpis(e, false)} naUsun={pytajUsun}>
          {#snippet pusto()}
            <Pusto ikona="kosz" tytul="Kosz jest pusty"
              opis="Usunięte pliki i foldery trafiają tutaj. Po {DNI_AUTOMATYCZNEGO_OPROZNIANIA} dniach serwer usuwa je trwale.">
              <Przycisk ikona="folder" onclick={() => idz('pliki')}>Przejdź do plików</Przycisk>
            </Pusto>
          {/snippet}
        </TabelaKosza>
      {/if}
    </Karta>
    <p class="opis uwaga">Wpisy starsze niż {DNI_AUTOMATYCZNEGO_OPROZNIANIA} dni serwer usuwa trwale raz na dobę.</p>
  {/if}
</div>

<OknoPotwierdzenia tytul="Miejsce jest zajęte" bind:otwarte={oknoNadpisz} napis="Przywróć i nadpisz" ikona="odswiez"
  naPotwierdz={przywrocZNadpisaniem}>
  <p>W <span class="mono">{wybrany?.sciezka}</span> jest już inny element.</p>
  <p class="opis">Obecny element trafi do kosza, a na jego miejsce wróci wersja z kosza.</p>
</OknoPotwierdzenia>

<OknoPotwierdzenia tytul="Usunąć trwale?" bind:otwarte={oknoUsun} napis="Usuń trwale" naPotwierdz={usunWybrany}>
  <p><span class="mono">{wybrany?.sciezka}</span>{wybrany?.folder ? ' (folder z całą zawartością)' : ''} zniknie z dysku.</p>
  <p class="opis">Tej operacji nie można cofnąć.</p>
</OknoPotwierdzenia>

<OknoPotwierdzenia tytul="Opróżnić kosz?" bind:otwarte={oknoOproznij} napis="Opróżnij kosz" naPotwierdz={oproznijKosz}>
  <p>Wszystkie elementy z kosza udziału <span class="mono">{udzial}</span> ({wpisy.length}, {rozmiar(suma)}) zostaną
    usunięte z dysku.</p>
  <p class="opis">Tej operacji nie można cofnąć.</p>
</OknoPotwierdzenia>

<style>
  .strona {
    display: flex;
    flex-direction: column;
    gap: var(--odstep-7);
    min-width: 0;
  }
  .wybor-udzialu {
    min-width: var(--kolumna-l);
  }
  .szkielet {
    padding: var(--odstep-l);
  }
  .uwaga {
    margin: 0;
  }
</style>
