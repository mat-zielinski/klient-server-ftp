package miniatury

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"testing"
)

// segmentExif: APP1 z TIFF (kolejność bajtów bo) i jednym wpisem IFD0 (tag, typ, wartość).
func segmentExif(bo binary.ByteOrder, tag, typ, wartosc uint16) []byte {
	var tiff bytes.Buffer
	if bo == binary.LittleEndian {
		tiff.WriteString("II")
	} else {
		tiff.WriteString("MM")
	}
	_ = binary.Write(&tiff, bo, uint16(42)) // celowo: zapis do bufora w pamięci nie zwraca błędu
	_ = binary.Write(&tiff, bo, uint32(8))  // celowo: zapis do bufora w pamięci nie zwraca błędu
	_ = binary.Write(&tiff, bo, uint16(1))  // celowo: zapis do bufora w pamięci nie zwraca błędu
	_ = binary.Write(&tiff, bo, tag)        // celowo: zapis do bufora w pamięci nie zwraca błędu
	_ = binary.Write(&tiff, bo, typ)        // celowo: zapis do bufora w pamięci nie zwraca błędu
	_ = binary.Write(&tiff, bo, uint32(1))  // celowo: zapis do bufora w pamięci nie zwraca błędu
	_ = binary.Write(&tiff, bo, wartosc)    // celowo: zapis do bufora w pamięci nie zwraca błędu
	_ = binary.Write(&tiff, bo, uint16(0))  // celowo: zapis do bufora w pamięci nie zwraca błędu
	_ = binary.Write(&tiff, bo, uint32(0))  // celowo: zapis do bufora w pamięci nie zwraca błędu
	tresc := append([]byte(naglowekExif), tiff.Bytes()...)
	seg := []byte{0xFF, znacznikAPP1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(tresc)+2))
	return append(seg, tresc...)
}

// zExif wstawia segment zaraz po SOI.
func zExif(jpg, seg []byte) []byte {
	return append(append(append([]byte{}, jpg[:2]...), seg...), jpg[2:]...)
}

func TestOrientacjaEXIF(t *testing.T) {
	jpg := []byte{0xFF, znacznikSOI, 0xFF, 0xD9}
	przypadki := []struct {
		nazwa string
		dane  []byte
		chce  int
	}{
		{"brak EXIF", jpg, 1},
		{"II 6", zExif(jpg, segmentExif(binary.LittleEndian, tagOrientacji, typSHORT, 6)), 6},
		{"MM 8", zExif(jpg, segmentExif(binary.BigEndian, tagOrientacji, typSHORT, 8)), 8},
		{"MM 3", zExif(jpg, segmentExif(binary.BigEndian, tagOrientacji, typSHORT, 3)), 3},
		{"wartość spoza 1..8", zExif(jpg, segmentExif(binary.BigEndian, tagOrientacji, typSHORT, 9)), 1},
		{"zły typ", zExif(jpg, segmentExif(binary.BigEndian, tagOrientacji, 4, 6)), 1},
		{"inny tag", zExif(jpg, segmentExif(binary.BigEndian, 0x010F, typSHORT, 6)), 1},
		{"nie JPEG", []byte("\x89PNG...."), 1},
		{"pusty", nil, 1},
		{"ucięty segment", zExif(jpg, segmentExif(binary.BigEndian, tagOrientacji, typSHORT, 6))[:20], 1},
		{"długość segmentu za plikiem", []byte{0xFF, znacznikSOI, 0xFF, znacznikAPP1, 0xFF, 0xFF, 'E'}, 1},
		{"długość 0", []byte{0xFF, znacznikSOI, 0xFF, znacznikAPP1, 0, 0, 0, 0}, 1},
	}
	for _, p := range przypadki {
		if o := orientacjaEXIF(bytes.NewReader(p.dane)); o != p.chce {
			t.Errorf("%s: %d, chcę %d", p.nazwa, o, p.chce)
		}
	}
}

// Każda orientacja: gdzie trafia lewy górny piksel obrazu 3 × 2 i jakie wymiary ma wynik.
func TestObroc(t *testing.T) {
	przypadki := []struct {
		orientacja, sz, wys int
		lgX, lgY            int
	}{
		{1, 3, 2, 0, 0}, {2, 3, 2, 2, 0}, {3, 3, 2, 2, 1}, {4, 3, 2, 0, 1},
		{5, 2, 3, 0, 0}, {6, 2, 3, 1, 0}, {7, 2, 3, 1, 2}, {8, 2, 3, 0, 2},
		{0, 3, 2, 0, 0}, {9, 3, 2, 0, 0},
	}
	for _, p := range przypadki {
		src := image.NewRGBA(image.Rect(0, 0, 3, 2))
		src.SetRGBA(0, 0, czerwony)
		dst := obroc(src, p.orientacja)
		if b := dst.Bounds(); b.Dx() != p.sz || b.Dy() != p.wys {
			t.Errorf("orientacja %d: %v", p.orientacja, b)
			continue
		}
		if dst.RGBAAt(p.lgX, p.lgY) != czerwony {
			t.Errorf("orientacja %d: lewy górny piksel nie w (%d,%d)", p.orientacja, p.lgX, p.lgY)
		}
	}
}

// Zdjęcie z telefonu trzymanego pionowo: zapisane poziomo (40 × 20) z orientacją 6 = miniatura pionowa.
func TestMiniaturaZOrientacja(t *testing.T) {
	s := noweSrodowisko(t)
	jpg := kodujJPEG(t, obrazPol(40, 20))
	s.zapisz(t, "telefon.jpg", zExif(jpg, segmentExif(binary.BigEndian, tagOrientacji, typSHORT, 6)))
	s.zapisz(t, "zwykle.jpg", jpg)
	przypadki := []struct {
		sciezka   string
		sz, wys   int
		gora, dol image.Point
	}{
		// Obrót 90° w prawo: lewa (czerwona) połowa ląduje na górze.
		{"/telefon.jpg", 20, 40, image.Pt(10, 2), image.Pt(10, 37)},
		{"/zwykle.jpg", 40, 20, image.Pt(2, 10), image.Pt(37, 10)},
	}
	for _, p := range przypadki {
		plik, err := s.m.Miniatura(context.Background(), "dom", p.sciezka, "s")
		if err != nil {
			t.Fatal(err)
		}
		img := wczytaj(t, plik)
		if b := img.Bounds(); b.Dx() != p.sz || b.Dy() != p.wys {
			t.Errorf("%s: %v, chcę %dx%d", p.sciezka, b, p.sz, p.wys)
			continue
		}
		if !podobny(img.At(p.gora.X, p.gora.Y), czerwony) || !podobny(img.At(p.dol.X, p.dol.Y), niebieski) {
			t.Errorf("%s: zła orientacja kolorów", p.sciezka)
		}
	}
}
