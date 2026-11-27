package miniatury

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/magazyn"
	"skrytka/wewn/rdzen"
)

var (
	czerwony  = color.RGBA{255, 0, 0, 255}
	niebieski = color.RGBA{0, 0, 255, 255}
)

// srodowisko: udział "dom" w t.TempDir i cache w osobnym t.TempDir.
type srodowisko struct {
	kat string
	m   *Miniatury
}

func noweSrodowisko(t *testing.T, opcje ...Opcja) *srodowisko {
	t.Helper()
	kat := t.TempDir()
	mag, err := magazyn.Nowy([]konfiguracja.Udzial{{Nazwa: "dom", Sciezka: kat}})
	if err != nil {
		t.Fatal(err)
	}
	return &srodowisko{kat: kat, m: Nowe(mag, filepath.Join(t.TempDir(), "miniatury"), opcje...)}
}

func (s *srodowisko) zapisz(t *testing.T, nazwa string, dane []byte) {
	t.Helper()
	p := filepath.Join(s.kat, filepath.FromSlash(nazwa))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, dane, 0o644); err != nil {
		t.Fatal(err)
	}
}

// obrazPol: lewa połowa czerwona, prawa niebieska.
func obrazPol(sz, wys int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, sz, wys))
	for y := range wys {
		for x := range sz {
			c := czerwony
			if x >= sz/2 {
				c = niebieski
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func kodujPNG(t *testing.T, img image.Image) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func kodujJPEG(t *testing.T, img image.Image) []byte {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func kodujGIF(t *testing.T, img image.Image) []byte {
	var b bytes.Buffer
	if err := gif.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// wczytaj miniaturę (musi być JPEG).
func wczytaj(t *testing.T, plik string) image.Image {
	t.Helper()
	f, err := os.Open(plik)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := jpeg.Decode(f)
	if err != nil {
		t.Fatalf("miniatura to nie JPEG: %v", err)
	}
	return img
}

// podobny: kolor zbliżony (JPEG jest stratny).
func podobny(c color.Color, chce color.RGBA) bool {
	r, g, b, _ := c.RGBA()
	odl := func(a uint32, b uint8) bool { d := int(a>>8) - int(b); return d > -60 && d < 60 }
	return odl(r, chce.R) && odl(g, chce.G) && odl(b, chce.B)
}

func TestMiniaturaFormaty(t *testing.T) {
	s := noweSrodowisko(t)
	s.zapisz(t, "a.png", kodujPNG(t, obrazPol(1000, 500)))
	s.zapisz(t, "zdjęcia/b.JPG", kodujJPEG(t, obrazPol(300, 600)))
	s.zapisz(t, "c.gif", kodujGIF(t, obrazPol(100, 50)))
	przypadki := []struct {
		sciezka, rozmiar string
		sz, wys          int
	}{
		{"/a.png", "s", 160, 80},
		{"/a.png", "m", 480, 240},
		{"/zdjęcia/b.JPG", "m", 240, 480},
		{"/zdjęcia/b.JPG", "s", 80, 160},
		{"/c.gif", "s", 100, 50}, // mniejszy niż wymiar: bez powiększania
	}
	for _, p := range przypadki {
		plik, err := s.m.Miniatura(context.Background(), "dom", p.sciezka, p.rozmiar)
		if err != nil {
			t.Errorf("%s %s: %v", p.sciezka, p.rozmiar, err)
			continue
		}
		img := wczytaj(t, plik)
		b := img.Bounds()
		if b.Dx() != p.sz || b.Dy() != p.wys {
			t.Errorf("%s %s: %dx%d, chcę %dx%d", p.sciezka, p.rozmiar, b.Dx(), b.Dy(), p.sz, p.wys)
		}
		if !podobny(img.At(1, b.Dy()/2), czerwony) || !podobny(img.At(b.Dx()-2, b.Dy()/2), niebieski) {
			t.Errorf("%s: złe kolory po skalowaniu", p.sciezka)
		}
	}
}

func TestMiniaturaBledy(t *testing.T) {
	s := noweSrodowisko(t)
	s.zapisz(t, "tekst.txt", []byte("abc"))
	s.zapisz(t, "obraz.bmp", []byte("BM"))
	s.zapisz(t, "zepsuty.png", []byte("to nie png"))
	s.zapisz(t, "folder.png/x.txt", []byte("x"))
	s.zapisz(t, "ok.png", kodujPNG(t, obrazPol(4, 4)))
	przypadki := []struct {
		sciezka, rozmiar string
		chce             error
	}{
		{"/ok.png", "xl", ErrZlyRozmiar},
		{"/ok.png", "", ErrZlyRozmiar},
		{"/tekst.txt", "s", ErrNieObraz},
		{"/folder.png", "s", ErrNieObraz},
		{"/obraz.bmp", "s", ErrNieObslugiwany},
		{"/zepsuty.png", "s", ErrUszkodzony},
		{"/brak.png", "s", rdzen.ErrNieZnaleziono},
		{"../poza.png", "s", rdzen.ErrPozaUdzialem},
	}
	for _, p := range przypadki {
		if _, err := s.m.Miniatura(context.Background(), "dom", p.sciezka, p.rozmiar); !errors.Is(err, p.chce) {
			t.Errorf("%s %q: %v, chcę %v", p.sciezka, p.rozmiar, err, p.chce)
		}
	}
	if _, err := s.m.Miniatura(context.Background(), "obcy", "/ok.png", "s"); !errors.Is(err, rdzen.ErrNieZnaleziono) {
		t.Errorf("obcy udział: %v", err)
	}
}

// pngZNaglowkiem: poprawny nagłówek PNG (IHDR z CRC) zapowiadający ogromny obraz, bez danych obrazu.
func pngZNaglowkiem(sz, wys uint32) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], sz)
	binary.BigEndian.PutUint32(ihdr[4:], wys)
	ihdr[8], ihdr[9] = 8, 2                                   // 8 bitów, RGB
	_ = binary.Write(&b, binary.BigEndian, uint32(len(ihdr))) // celowo: zapis do bufora w pamięci nie zwraca błędu
	typIDane := append([]byte("IHDR"), ihdr...)
	b.Write(typIDane)
	_ = binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(typIDane)) // celowo: zapis do bufora w pamięci nie zwraca błędu
	return b.Bytes()
}

func TestBombaDekompresyjna(t *testing.T) {
	s := noweSrodowisko(t)
	s.zapisz(t, "bomba.png", pngZNaglowkiem(20000, 20000)) // 400 Mpx > limit
	s.zapisz(t, "maly.png", kodujPNG(t, obrazPol(100, 100)))
	if _, err := s.m.Miniatura(context.Background(), "dom", "/bomba.png", "s"); !errors.Is(err, ErrZaDuzy) {
		t.Errorf("bomba: %v, chcę ErrZaDuzy", err)
	}
	przypadki := []struct {
		bajty, piksele int64
		chce           error
	}{
		{1 << 20, 100*100 - 1, ErrZaDuzy},
		{10, 1 << 20, ErrZaDuzy}, // plik większy niż limit bajtów
		{1 << 20, 100 * 100, nil},
	}
	for _, p := range przypadki {
		s.m.maksBajtow, s.m.maksPikseli = p.bajty, p.piksele
		if _, err := s.m.Miniatura(context.Background(), "dom", "/maly.png", "m"); !errors.Is(err, p.chce) {
			t.Errorf("limit %d B / %d px: %v, chcę %v", p.bajty, p.piksele, err, p.chce)
		}
	}
}

func TestCacheTrafienie(t *testing.T) {
	s := noweSrodowisko(t)
	s.zapisz(t, "a.png", kodujPNG(t, obrazPol(500, 500)))
	ctx := context.Background()
	p1, err := s.m.Miniatura(ctx, "dom", "/a.png", "s")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(p1) != s.m.katalog || filepath.Ext(p1) != ".jpg" || len(filepath.Base(p1)) != 64+4 {
		t.Fatalf("zła ścieżka cache %s", p1)
	}
	// Podmieniamy plik w cache: trafienie zwraca go bez ponownego skalowania.
	if err := os.WriteFile(p1, []byte("ZNACZNIK"), 0o644); err != nil {
		t.Fatal(err)
	}
	p2, err := s.m.Miniatura(ctx, "dom", "/a.png", "s")
	if dane, _ := os.ReadFile(p2); err != nil || p2 != p1 || string(dane) != "ZNACZNIK" {
		t.Fatalf("brak trafienia: %v %s", err, p2)
	}
	pm, err := s.m.Miniatura(ctx, "dom", "/a.png", "m")
	if err != nil || pm == p1 {
		t.Fatalf("rozmiar m z tym samym kluczem: %v", err)
	}
	// Zmieniony plik źródłowy (inny rozmiar i data) = nowy klucz.
	s.zapisz(t, "a.png", kodujPNG(t, obrazPol(300, 200)))
	if err := os.Chtimes(filepath.Join(s.kat, "a.png"), time.Now(), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	p3, err := s.m.Miniatura(ctx, "dom", "/a.png", "s")
	if err != nil || p3 == p1 {
		t.Fatalf("zmienione źródło dało stary plik: %v", err)
	}
	if b := wczytaj(t, p3).Bounds(); b.Dx() != 160 || b.Dy() != 106 {
		t.Errorf("nowa miniatura %v", b)
	}
}

func TestRownoczesneTworzenie(t *testing.T) {
	s := noweSrodowisko(t)
	s.zapisz(t, "a.png", kodujPNG(t, obrazPol(800, 600)))
	var wg sync.WaitGroup
	wyniki := make([]string, 8)
	bledy := make([]error, 8)
	for i := range wyniki {
		wg.Go(func() { wyniki[i], bledy[i] = s.m.Miniatura(context.Background(), "dom", "/a.png", "m") })
	}
	wg.Wait()
	for i := range wyniki {
		if bledy[i] != nil || wyniki[i] != wyniki[0] {
			t.Fatalf("%d: %v %s", i, bledy[i], wyniki[i])
		}
	}
	wpisy, _ := os.ReadDir(s.m.katalog)
	if len(wpisy) != 1 {
		t.Errorf("w cache %d plików (pliki tymczasowe?)", len(wpisy))
	}
}

func TestSprzataj(t *testing.T) {
	s := noweSrodowisko(t, ZLimitemCache(1000))
	if err := s.m.Sprzataj(); err != nil {
		t.Fatalf("brak folderu: %v", err)
	}
	if err := os.MkdirAll(s.m.katalog, 0o755); err != nil {
		t.Fatal(err)
	}
	baza := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	pliki := []struct {
		nazwa string
		bajty int
		wiek  time.Duration
	}{{"stary.jpg", 400, 3 * time.Hour}, {"sredni.jpg", 400, 2 * time.Hour}, {"nowy.jpg", 400, time.Hour}}
	for _, p := range pliki {
		sc := filepath.Join(s.m.katalog, p.nazwa)
		if err := os.WriteFile(sc, make([]byte, p.bajty), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(sc, baza.Add(-p.wiek), baza.Add(-p.wiek)); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.m.Sprzataj(); err != nil {
		t.Fatal(err)
	}
	// 1200 B > 1000: usuwamy najstarsze, aż ≤ 900 (90 %): znika tylko stary.jpg.
	for _, p := range pliki {
		_, err := os.Stat(filepath.Join(s.m.katalog, p.nazwa))
		if zostal, chce := err == nil, p.nazwa != "stary.jpg"; zostal != chce {
			t.Errorf("%s: został=%v, chcę %v", p.nazwa, zostal, chce)
		}
	}
	// Poniżej limitu nic nie znika.
	if err := s.m.Sprzataj(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.m.katalog, "nowy.jpg")); err != nil {
		t.Error("usunięto plik poniżej limitu")
	}
}
