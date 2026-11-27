package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	nethttp "net/http"
	"runtime/debug"
	"strings"
	"unicode/utf8"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
	"skrytka/wewn/uslugi/dziennik"
)

// kluczKontekstu: typ prywatny, żeby nikt spoza pakietu nie nadpisał wartości.
type kluczKontekstu int

const (
	kluczUzytkownika kluczKontekstu = iota
	kluczTokenu
)

// naglowkiBezpieczenstwa ustawia nagłówki obowiązujące dla każdej odpowiedzi.
func naglowkiBezpieczenstwa(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", konfiguracja.PolitykaCSP)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", konfiguracja.PolitykaReferrer)
		h.Set("X-Frame-Options", "SAMEORIGIN")
		next.ServeHTTP(w, r)
	})
}

// odzyskiwanie łapie panic w obsłudze: wpis w dzienniku i 500 zamiast zerwanego połączenia.
func (s *serwer) odzyskiwanie(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		defer func() {
			p := recover()
			if p == nil {
				return
			}
			if p == nethttp.ErrAbortHandler { //nolint:errorlint // celowo: porównanie wartości panic
				panic(p)
			}
			s.dziennik.Zapisz(dziennik.Wpis{
				Czas: s.teraz(), Rodzaj: dziennik.RodzajSerwer, Kto: loginZ(r), IP: adresIP(r),
				Protokol: protokol(r), Wynik: dziennik.WynikBlad,
				Opis: fmt.Sprintf("Panika w %s %s: %v\n%s", r.Method, r.URL.Path, p, debug.Stack()),
			})
			piszBlad(w, nethttp.StatusInternalServerError, "Błąd serwera. Szczegóły w dzienniku.")
		}()
		next.ServeHTTP(w, r)
	})
}

// czyZmieniaStan: metody wymagające nagłówka CSRF.
func czyZmieniaStan(metoda string) bool {
	switch metoda {
	case nethttp.MethodGet, nethttp.MethodHead, nethttp.MethodOptions:
		return false
	}
	return true
}

// ochronaCSRF: każde żądanie zmieniające stan musi mieć X-Skrytka: 1 (formularz z innej strony go nie ustawi).
func ochronaCSRF(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if czyZmieniaStan(r.Method) && r.Header.Get(konfiguracja.NaglowekCSRF) != konfiguracja.WartoscCSRF {
			kod, komunikat := kodBledu(errCSRF)
			piszBlad(w, kod, komunikat)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// sesjaWKontekscie: użytkownik z ciasteczka sesji trafia do kontekstu żądania (brak = brak wpisu).
func (s *serwer) sesjaWKontekscie(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		c, err := r.Cookie(konfiguracja.NazwaCiasteczkaSesji)
		if err == nil && c.Value != "" {
			if u, ok := s.sesje.Sprawdz(c.Value); ok {
				ctx := context.WithValue(r.Context(), kluczUzytkownika, u)
				ctx = context.WithValue(ctx, kluczTokenu, c.Value)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// uzytkownikZ zwraca zalogowanego użytkownika z kontekstu.
func uzytkownikZ(r *nethttp.Request) (rdzen.Uzytkownik, bool) {
	u, ok := r.Context().Value(kluczUzytkownika).(rdzen.Uzytkownik)
	return u, ok
}

func tokenZ(r *nethttp.Request) string {
	t, _ := r.Context().Value(kluczTokenu).(string) // celowo: brak = pusty token
	return t
}

func loginZ(r *nethttp.Request) string {
	u, _ := uzytkownikZ(r) // celowo: brak sesji = pusty login
	return u.Login
}

// obslugaUzytkownika: obsługa z gwarantowanym zalogowanym użytkownikiem.
type obslugaUzytkownika func(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik)

// zalogowany: bez sesji 401.
func (s *serwer) zalogowany(h obslugaUzytkownika) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		u, ok := uzytkownikZ(r)
		if !ok {
			s.blad(w, r, errBrakSesji)
			return
		}
		h(w, r, u)
	}
}

// tylkoAdmin: bez sesji 401, bez prawa administracji 403 (przez bramkę rdzenia).
func (s *serwer) tylkoAdmin(h obslugaUzytkownika) nethttp.HandlerFunc {
	return s.zalogowany(func(w nethttp.ResponseWriter, r *nethttp.Request, u rdzen.Uzytkownik) {
		if err := rdzen.CzyWolno(u, rdzen.OpAdministracja, "", false); err != nil {
			s.blad(w, r, err)
			return
		}
		h(w, r, u)
	})
}

// czyAdmin: pomocnicze do filtrów „admin wszystko, inni swoje”.
func czyAdmin(u rdzen.Uzytkownik) bool {
	return rdzen.CzyWolno(u, rdzen.OpAdministracja, "", false) == nil
}

// czytajJSON dekoduje treść z limitem rozmiaru MaksCialoJSON.
func czytajJSON(w nethttp.ResponseWriter, r *nethttp.Request, v any) error {
	r.Body = nethttp.MaxBytesReader(w, r.Body, konfiguracja.MaksCialoJSON)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var za *nethttp.MaxBytesError
		if errors.As(err, &za) {
			return errZaDuze
		}
		return errZlyJSON
	}
	return nil
}

// adresIP klienta z połączenia TCP (nagłówkom X-Forwarded-For nie ufamy).
func adresIP(r *nethttp.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// protokol żądania do dziennika i transferów.
func protokol(r *nethttp.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// przytnij tekst z zewnątrz do dziennika (bajty, bez rozcinania znaku UTF-8).
func przytnij(s string, maks int) string {
	if len(s) <= maks {
		return s
	}
	s = s[:maks]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

// wpisDziennika: wspólne pola wpisu z żądania.
func (s *serwer) wpisDziennika(r *nethttp.Request, rodzaj, kto string) dziennik.Wpis {
	return dziennik.Wpis{Czas: s.teraz(), Rodzaj: rodzaj, Kto: kto, IP: adresIP(r), Protokol: protokol(r)}
}

// tekstBezSterujacych: do nazw plików w nagłówkach.
func tekstBezSterujacych(s string) string {
	return strings.Map(func(c rune) rune {
		if c < 0x20 || c == 0x7f {
			return '_'
		}
		return c
	}, s)
}
