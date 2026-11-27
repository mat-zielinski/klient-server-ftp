package udostepnienia

import (
	"encoding/json"
	"errors"
	"os"
	"sort"

	"skrytka/wewn/konfiguracja"
)

// wczytaj czyta plik linków; brak pliku = pusta mapa.
func wczytaj(plik string) (map[string]Link, error) {
	linki := map[string]Link{}
	dane, err := os.ReadFile(plik)
	if errors.Is(err, os.ErrNotExist) {
		return linki, nil
	}
	if err != nil {
		return nil, err
	}
	var lista []Link
	if err := json.Unmarshal(dane, &lista); err != nil {
		return nil, err
	}
	for _, l := range lista {
		if l.ID == "" {
			continue
		}
		l.MaHaslo = l.MaHaslo || l.HashHasla != "" // niespójny wpis = zamknięty (hasło nie przejdzie)
		linki[l.ID] = l
	}
	return linki, nil
}

// zapisz zapisuje wszystkie linki (z hashami haseł) atomowo, w stałej kolejności.
func zapisz(plik string, linki map[string]Link) error {
	lista := make([]Link, 0, len(linki))
	for _, l := range linki {
		lista = append(lista, l)
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].ID < lista[j].ID })
	dane, err := json.MarshalIndent(lista, "", "  ")
	if err != nil {
		return err
	}
	return konfiguracja.ZapiszAtomowo(plik, dane)
}
