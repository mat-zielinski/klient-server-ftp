package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	nethttp "net/http"
	"strconv"
	"strings"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/udostepnienia"
)

// nowyKluczLinkow: losowy klucz HMAC (budowa serwera; brak losowości systemu = nie da się bezpiecznie ruszyć).
func nowyKluczLinkow() []byte {
	k := make([]byte, konfiguracja.DlugoscKluczaLinkow)
	if _, err := rand.Read(k); err != nil {
		panic("Brak losowości systemu dla klucza linków: " + err.Error())
	}
	return k
}

// podpisLinku: HMAC-SHA256(klucz, id + "|" + czas wygaśnięcia w sekundach Unix), base64url.
func (s *serwer) podpisLinku(id string, wygasa int64) string {
	m := hmac.New(sha256.New, s.kluczLinkow)
	m.Write([]byte(id + "|" + strconv.FormatInt(wygasa, 10))) // celowo: hash.Write nigdy nie zwraca błędu
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// ustawCiasteczkoLinku (po poprawnym haśle): HttpOnly, SameSite=Strict, ścieżka tylko tego linku,
// ważne WaznoscCiasteczkaLinku, ale nie dłużej niż sam link. Wartość: "<wygasa>.<podpis>".
func (s *serwer) ustawCiasteczkoLinku(w nethttp.ResponseWriter, r *nethttp.Request, l udostepnienia.Link) {
	teraz := s.teraz()
	wygasa := teraz.Add(konfiguracja.WaznoscCiasteczkaLinku)
	if !l.Wygasa.IsZero() && l.Wygasa.Before(wygasa) {
		wygasa = l.Wygasa
	}
	unix := wygasa.Unix()
	wiek := time.Unix(unix, 0).Sub(teraz)
	if wiek <= 0 {
		return
	}
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name: konfiguracja.NazwaCiasteczkaLinku, Value: strconv.FormatInt(unix, 10) + "." + s.podpisLinku(l.ID, unix),
		Path: konfiguracja.PrefiksAPILinku + l.ID, Expires: time.Unix(unix, 0), MaxAge: int(wiek / time.Second),
		HttpOnly: true, SameSite: nethttp.SameSiteStrictMode, Secure: r.TLS != nil,
	})
}

// ciasteczkoLinkuWazne: którekolwiek ciasteczko linku ma poprawny podpis dla TEGO id i nie wygasło.
func (s *serwer) ciasteczkoLinkuWazne(r *nethttp.Request, id string) bool {
	teraz := s.teraz()
	for _, c := range r.CookiesNamed(konfiguracja.NazwaCiasteczkaLinku) {
		czas, podpis, ok := strings.Cut(c.Value, ".")
		if !ok {
			continue
		}
		unix, err := strconv.ParseInt(czas, 10, 64)
		if err != nil {
			continue
		}
		wygasa := time.Unix(unix, 0)
		// Wygasłe albo dalej niż pozwala ważność (podpis i tak by nie pasował, to tylko tania kontrola).
		if !teraz.Before(wygasa) || wygasa.After(teraz.Add(konfiguracja.WaznoscCiasteczkaLinku)) {
			continue
		}
		if hmac.Equal([]byte(podpis), []byte(s.podpisLinku(id, unix))) {
			return true
		}
	}
	return false
}

// linkBezHasla: link otwierany ciasteczkiem albo bez podanego hasła (bez liczenia próby hasła).
// Wygasły albo wyczerpany = ErrWygaslo; link z hasłem bez ciasteczka = ErrZleDane (401, prośba o hasło).
func (s *serwer) linkBezHasla(id string, zCiasteczkiem bool) (udostepnienia.Link, error) {
	l, err := s.udostepnienia.Pobierz(id)
	if err != nil {
		return l, err
	}
	teraz := s.teraz()
	if (!l.Wygasa.IsZero() && !teraz.Before(l.Wygasa)) || (l.LimitPobran > 0 && l.Pobrano >= l.LimitPobran) {
		return udostepnienia.Link{}, rdzen.ErrWygaslo
	}
	if l.MaHaslo && !zCiasteczkiem {
		return udostepnienia.Link{}, rdzen.ErrZleDane
	}
	return l, nil
}

// otworzLink: ważne ciasteczko ALBO hasło (X-Haslo / ?haslo). Drugi wynik: czy sprawdzono hasło
// (tylko wtedy wolno wydać nowe ciasteczko, żeby ciasteczko nie przedłużało się samo).
func (s *serwer) otworzLink(r *nethttp.Request) (udostepnienia.Link, bool, error) {
	id := r.PathValue("id")
	if s.ciasteczkoLinkuWazne(r, id) {
		l, err := s.linkBezHasla(id, true)
		return l, false, err
	}
	haslo := hasloLinku(r)
	if haslo == "" {
		l, err := s.linkBezHasla(id, false)
		if err == nil {
			// Link bez hasła: zwykłe sprawdzenie usługi (ta sama ścieżka co dotąd).
			l, err = s.udostepnienia.Sprawdz(id, "")
		}
		return l, false, err
	}
	l, err := s.udostepnienia.Sprawdz(id, haslo)
	return l, err == nil && l.MaHaslo, err
}
