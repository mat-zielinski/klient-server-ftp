<script lang="ts">
  // Pasek postępu: wartosc 0..1 albo null (nieokreślony, przesuwający się pasek). `stan` barwi wypełnienie.
  interface Props {
    wartosc: number | null
    stan?: 'ok' | 'info' | 'blad' | 'ostrz' | 'wyl'
    etykieta?: string
  }

  let { wartosc, stan = 'info', etykieta = 'Postęp' }: Props = $props()

  const ulamek = $derived(wartosc === null ? null : Math.min(1, Math.max(0, wartosc)))
</script>

<div
  class="pasek"
  role="progressbar"
  aria-label={etykieta}
  aria-valuemin="0"
  aria-valuemax="100"
  aria-valuenow={ulamek === null ? undefined : Math.round(ulamek * 100)}
>
  <div
    class="wypelnienie {stan}"
    class:nieokreslony={ulamek === null}
    style:width={ulamek === null ? undefined : `${ulamek * 100}%`}
  ></div>
</div>

<style>
  .pasek {
    position: relative;
    width: 100%;
    height: var(--wysokosc-pasek-postepu);
    border-radius: var(--wysokosc-pasek-postepu);
    background: var(--kolor-karta);
    overflow: hidden;
  }
  .wypelnienie {
    height: 100%;
    border-radius: inherit;
    transition: width var(--czas-sredni) linear;
  }
  .info {
    background: var(--kolor-niebieski);
  }
  .ok {
    background: var(--kolor-zielony);
  }
  .blad {
    background: var(--kolor-czerwony);
  }
  .ostrz {
    background: var(--kolor-zolty);
  }
  .wyl {
    background: var(--kolor-opis);
  }
  .nieokreslony {
    position: absolute;
    width: 35%;
    animation: przesun 1.2s ease-in-out infinite;
  }
  @keyframes przesun {
    from {
      left: -35%;
    }
    to {
      left: 100%;
    }
  }
</style>
