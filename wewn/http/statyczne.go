package http

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"mime"
	nethttp "net/http"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"skrytka/wewn/konfiguracja"
)

// plikInterfejsu: zawartość, wersja skompresowana i ETag (pliki wkompilowane, więc stałe).
type plikInterfejsu struct {
	dane  []byte
	gz    []byte
	etag  string
	typ   string
	cache string
}

// pamiecStatycznych: raz wczytane pliki interfejsu (fs.FS z web/dist).
type pamiecStatycznych struct {
	zrodlo  fs.FS
	zHashem *regexp.Regexp
	mu      sync.Mutex
	pliki   map[string]*plikInterfejsu
}

func nowaPamiecStatycznych(zrodlo fs.FS) *pamiecStatycznych {
	return &pamiecStatycznych{
		zrodlo: zrodlo, zHashem: regexp.MustCompile(konfiguracja.WzorNazwyZHashem),
		pliki: map[string]*plikInterfejsu{},
	}
}

// wczytaj zwraca plik albo nil (brak, folder, brak źródła).
func (p *pamiecStatycznych) wczytaj(nazwa string) *plikInterfejsu {
	if p.zrodlo == nil || !fs.ValidPath(nazwa) {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if pl, ok := p.pliki[nazwa]; ok {
		return pl
	}
	info, err := fs.Stat(p.zrodlo, nazwa)
	if err != nil || info.IsDir() {
		return nil
	}
	dane, err := fs.ReadFile(p.zrodlo, nazwa)
	if err != nil {
		return nil
	}
	suma := sha256.Sum256(dane)
	pl := &plikInterfejsu{dane: dane, etag: `"` + hex.EncodeToString(suma[:8]) + `"`, cache: konfiguracja.CacheBez}
	pl.typ = typPlikuInterfejsu(nazwa)
	if pl.typ == "" {
		pl.typ = "application/octet-stream"
	}
	bezCache := nazwa == konfiguracja.PlikIndeksu || path.Base(nazwa) == konfiguracja.PlikServiceWorkera
	if !bezCache && p.zHashem.MatchString(path.Base(nazwa)) {
		pl.cache = konfiguracja.CacheZasobowZHashem
	}
	if slices.Contains(konfiguracja.RozszerzeniaGzip, strings.ToLower(path.Ext(nazwa))) {
		pl.gz = spakuj(dane)
	}
	p.pliki[nazwa] = pl
	return pl
}

// spakuj: gzip w pamięci (nil przy błędzie = wysyłamy bez kompresji).
func spakuj(dane []byte) []byte {
	var b bytes.Buffer
	z, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		return nil
	}
	if _, err := z.Write(dane); err != nil {
		return nil
	}
	if err := z.Close(); err != nil {
		return nil
	}
	return b.Bytes()
}

// przyjmujeGzip: nagłówek Accept-Encoding zawiera gzip (bez q=0).
func przyjmujeGzip(r *nethttp.Request) bool {
	for _, c := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		nazwa, param, _ := strings.Cut(strings.TrimSpace(c), ";")
		if strings.EqualFold(strings.TrimSpace(nazwa), "gzip") && strings.ReplaceAll(param, " ", "") != "q=0" {
			return true
		}
	}
	return false
}

// plikStatyczny: interfejs SPA; nieznana ścieżka = index.html, nieznany adres /api/ = 404 JSON.
func (s *serwer) plikStatyczny(w nethttp.ResponseWriter, r *nethttp.Request) {
	if strings.HasPrefix(r.URL.Path, konfiguracja.PrefiksAPI) {
		s.blad(w, r, errNieznanyAdres)
		return
	}
	if r.Method != nethttp.MethodGet && r.Method != nethttp.MethodHead {
		s.blad(w, r, errMetoda)
		return
	}
	nazwa := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	pl := s.statyczne.wczytaj(nazwa)
	if pl == nil {
		pl = s.statyczne.wczytaj(konfiguracja.PlikIndeksu)
		if pl == nil {
			piszBlad(w, nethttp.StatusNotFound, "Brak plików interfejsu.")
			return
		}
	}
	h := w.Header()
	h.Set("Content-Type", pl.typ)
	h.Set("Cache-Control", pl.cache)
	h.Set("Vary", "Accept-Encoding")
	dane, etag := pl.dane, pl.etag
	if pl.gz != nil && przyjmujeGzip(r) {
		h.Set("Content-Encoding", "gzip")
		dane, etag = pl.gz, strings.TrimSuffix(pl.etag, `"`)+`-gz"`
	}
	h.Set("ETag", etag)
	nethttp.ServeContent(w, r, "", time.Time{}, bytes.NewReader(dane))
}

// typPlikuInterfejsu: najpierw stała tablica (webmanifest, js), potem typy systemu.
func typPlikuInterfejsu(nazwa string) string {
	ext := strings.ToLower(path.Ext(nazwa))
	if t, ok := konfiguracja.TypyStatycznych[ext]; ok {
		return t
	}
	return mime.TypeByExtension(ext)
}
