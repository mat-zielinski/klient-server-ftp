<script lang="ts">
  // Tabela aktywnych transferów (Pulpit, admin): kto, IP, protokół, plik, postęp, prędkość, pozostało,
  // mini wykres prędkości i przycisk Przerwij (DELETE /api/transfery?id).
  import type { StanTransferu } from '../../api/typy'
  import { trescBledu } from '../../api/klient'
  import Tabela, { type KolumnaTabeli } from '../wspolne/Tabela.svelte'
  import Ikona from '../wspolne/Ikona.svelte'
  import PasekPostepu from '../wspolne/PasekPostepu.svelte'
  import Pigulka from '../wspolne/Pigulka.svelte'
  import Przycisk from '../wspolne/Przycisk.svelte'
  import Pusto from '../wspolne/Pusto.svelte'
  import MiniWykres from '../wykresy/MiniWykres.svelte'
  import { czasTrwania, predkosc, procent, rozmiar } from '../../stan/format'
  import { powiadom } from '../../stan/powiadomienia.svelte'
  import type { MonitorTransferow } from './monitor.svelte'
  import { nazwaPliku, stanProtokolu } from './typy'

  interface Props { monitor: MonitorTransferow }
  let { monitor }: Props = $props()

  const kolumny: KolumnaTabeli[] = [
    { klucz: 'kto', etykieta: 'Kto', szerokosc: 'm' },
    { klucz: 'ip', etykieta: 'IP', szerokosc: 'm', ukryjPonizej: 'waski' },
    { klucz: 'protokol', etykieta: 'Protokół', szerokosc: 's' },
    { klucz: 'plik', etykieta: 'Plik' },
    { klucz: 'postep', etykieta: 'Postęp', szerokosc: 'l' },
    { klucz: 'predkosc', etykieta: 'Prędkość', szerokosc: 'm', wyrownanie: 'prawo' },
    { klucz: 'pozostalo', etykieta: 'Pozostało', szerokosc: 's', wyrownanie: 'prawo', ukryjPonizej: 'sredni' },
    { klucz: 'wykres', etykieta: 'Ostatnie 60 s', szerokosc: 'm', ukryjPonizej: 'waski' },
    { klucz: 'akcje', etykieta: '', szerokosc: 'xs', wyrownanie: 'prawo' },
  ]

  let przerywane = $state<Set<string>>(new Set())

  function ulamek(t: StanTransferu): number | null {
    return t.rozmiar > 0 ? t.przeslano / t.rozmiar : null
  }

  async function przerwij(t: StanTransferu) {
    przerywane = new Set([...przerywane, t.id])
    try {
      await monitor.przerwij(t.id)
      powiadom(`Przerwano transfer „${nazwaPliku(t.sciezka)}”.`, 'ok')
    } catch (b) {
      powiadom(trescBledu(b), 'blad')
    } finally {
      przerywane = new Set([...przerywane].filter((id) => id !== t.id))
    }
  }
</script>

<Tabela {kolumny} wiersze={monitor.transfery}>
  {#snippet wiersz(t: StanTransferu)}
    {@const u = ulamek(t)}
    <tr>
      <td class="skroc" title={t.kto}>{t.kto || 'Gość'}</td>
      <td class="mono opis">{t.ip}</td>
      <td><Pigulka stan={stanProtokolu(t.protokol)}>{t.protokol.toUpperCase()}</Pigulka></td>
      <td title="{t.udzial}{t.sciezka}">
        <span class="plik">
          <span class="kierunek" title={t.kierunek === 'wgranie' ? 'Wgrywanie' : 'Pobieranie'}>
            <Ikona nazwa={t.kierunek === 'wgranie' ? 'wgraj' : 'pobierz'} rozmiar="s" />
          </span>
          <span class="skroc">{nazwaPliku(t.sciezka)}</span>
          <span class="opis skroc">{t.udzial}</span>
        </span>
      </td>
      <td>
        <span class="postep">
          <span class="pasek"><PasekPostepu wartosc={u} stan="info" etykieta="Postęp transferu" /></span>
          <span class="mono opis">{u === null ? rozmiar(t.przeslano) : procent(u)}</span>
        </span>
      </td>
      <td class="prawo mono">{predkosc(t.predkosc_bps)}</td>
      <td class="prawo mono opis">{t.pozostalo_s > 0 ? czasTrwania(t.pozostalo_s) : 'Brak'}</td>
      <td><MiniWykres wartosci={t.historia ?? []} opis="Prędkość z ostatnich 60 s" /></td>
      <td class="prawo">
        <Przycisk rodzaj="duch" rozmiar="maly" ikona="stop" tytul="Przerwij transfer"
          wylaczony={przerywane.has(t.id)} onclick={() => przerwij(t)} />
      </td>
    </tr>
  {/snippet}
  {#snippet pusto()}
    <Pusto ikona="aktywnosc" tytul={monitor.blad ? 'Nie udało się wczytać transferów' : 'Brak aktywnych transferów'}
      opis={monitor.blad || 'Pobrania i wgrania pojawią się tu na żywo.'} />
  {/snippet}
</Tabela>

<style>
  .plik {
    display: flex;
    align-items: center;
    gap: var(--odstep-2);
    min-width: 0;
  }
  .kierunek {
    display: inline-flex;
    color: var(--kolor-opis);
  }
  .plik .opis {
    flex: none;
    max-width: 40%;
  }
  .postep {
    display: flex;
    align-items: center;
    gap: var(--odstep-3);
  }
  .pasek {
    flex: 1;
    min-width: 0;
  }
  .postep .mono {
    min-width: 5ch;
    text-align: right;
  }
</style>
