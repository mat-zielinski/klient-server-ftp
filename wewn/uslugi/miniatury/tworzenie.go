package miniatury

import (
	"context"
	"image"
	"image/color"
	_ "image/gif" // celowo: rejestracja dekodera GIF
	"image/jpeg"
	_ "image/png" // celowo: rejestracja dekodera PNG
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // celowo: rejestracja dekodera WebP

	"skrytka/wewn/konfiguracja"
	"skrytka/wewn/rdzen"
)

// utworz dekoduje źródło (pod semaforem), skaluje, obraca wg EXIF i zapisuje JPEG atomowo.
func (x *Miniatury) utworz(ctx context.Context, udzial string, w rdzen.Wpis, bok int, plik string) error {
	select {
	case x.semafor <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-x.semafor }()
	img, orientacja, err := x.dekoduj(udzial, w.Sciezka)
	if err != nil {
		return err
	}
	return zapiszJPEG(plik, obroc(skaluj(img, bok), orientacja))
}

// dekoduj: nagłówek (image.DecodeConfig) przed pełnym dekodowaniem; za dużo pikseli = ErrZaDuzy.
func (x *Miniatury) dekoduj(udzial, sciezka string) (image.Image, int, error) {
	f, _, err := x.m.Otworz(udzial, sciezka)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close() // celowo: tylko odczyt
	zrodlo := io.LimitReader(f, x.maksBajtow)
	cfg, format, err := image.DecodeConfig(zrodlo)
	if err != nil {
		return nil, 0, ErrUszkodzony
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > x.maksPikseli {
		return nil, 0, ErrZaDuzy
	}
	orientacja := 1
	if format == "jpeg" {
		orientacja = orientacjaEXIF(io.NewSectionReader(f, 0, konfiguracja.MiniaturaMaksBajtowExif))
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, 0, err
	}
	img, _, err := image.Decode(io.LimitReader(f, x.maksBajtow))
	if err != nil {
		return nil, 0, ErrUszkodzony
	}
	return img, orientacja, nil
}

// skaluj do dłuższego boku bok (bez powiększania) filtrem CatmullRom na białym tle (przezroczystość PNG/GIF).
func skaluj(src image.Image, bok int) *image.RGBA {
	b := src.Bounds()
	sz, wys := b.Dx(), b.Dy()
	if sz > bok || wys > bok {
		if sz >= wys {
			sz, wys = bok, max(1, wys*bok/b.Dx())
		} else {
			sz, wys = max(1, sz*bok/b.Dy()), bok
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, sz, wys))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

// zapiszJPEG: plik tymczasowy w tym samym folderze, potem zamiana (czytelnik nie zobaczy połowy pliku).
func zapiszJPEG(plik string, img image.Image) error {
	kat := filepath.Dir(plik)
	if err := os.MkdirAll(kat, konfiguracja.MiniaturaUprawnieniaFolderu); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(kat, ".tmp_*")
	if err != nil {
		return err
	}
	err = jpeg.Encode(tmp, img, &jpeg.Options{Quality: konfiguracja.MiniaturaJakoscJPEG})
	if errZ := tmp.Close(); err == nil {
		err = errZ
	}
	if err == nil {
		err = os.Rename(tmp.Name(), plik)
		if err != nil && istnieje(plik) {
			err = nil // Windows: równoległy czytelnik trzyma gotową miniaturę o tym samym kluczu
		}
	}
	if err != nil {
		_ = os.Remove(tmp.Name()) // celowo: sprzątanie po nieudanym zapisie, ważniejszy jest err
	}
	return err
}

// touch ustawia czas pliku na teraz (ostatnie użycie miniatury).
func touch(plik string) error {
	teraz := time.Now()
	return os.Chtimes(plik, teraz, teraz)
}

// istnieje: zwykły plik pod ścieżką.
func istnieje(plik string) bool {
	fi, err := os.Stat(plik)
	return err == nil && fi.Mode().IsRegular()
}
