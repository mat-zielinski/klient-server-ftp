package miniatury

import (
	"encoding/binary"
	"image"
	"io"
)

// Znaczniki JPEG i EXIF używane przy odczycie orientacji.
const (
	znacznikSOI     = 0xD8
	znacznikAPP1    = 0xE1
	znacznikSOS     = 0xDA
	tagOrientacji   = 0x0112
	typSHORT        = 3
	dlugoscWpisuIFD = 12
	naglowekExif    = "Exif\x00\x00"
)

// orientacjaEXIF czyta tag 0x0112 z segmentu APP1 pliku JPEG (1..8); brak albo błąd = 1 (bez obrotu).
func orientacjaEXIF(r io.Reader) int {
	dane, err := io.ReadAll(r)
	if err != nil || len(dane) < 4 || dane[0] != 0xFF || dane[1] != znacznikSOI {
		return 1
	}
	for i := 2; i+4 <= len(dane); {
		if dane[i] != 0xFF {
			return 1
		}
		znacznik := dane[i+1]
		if znacznik == 0xFF { // wypełnienie między segmentami
			i++
			continue
		}
		if znacznik == znacznikSOS {
			return 1
		}
		dl := int(binary.BigEndian.Uint16(dane[i+2:]))
		if dl < 2 || i+2+dl > len(dane) {
			return 1
		}
		seg := dane[i+4 : i+2+dl]
		if znacznik == znacznikAPP1 && len(seg) > len(naglowekExif) && string(seg[:len(naglowekExif)]) == naglowekExif {
			return orientacjaTIFF(seg[len(naglowekExif):])
		}
		i += 2 + dl
	}
	return 1
}

// orientacjaTIFF: nagłówek TIFF (II/MM), IFD0, wpis 0x0112 typu SHORT.
func orientacjaTIFF(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	ifd := int64(bo.Uint32(t[4:]))
	if ifd < 8 || ifd+2 > int64(len(t)) {
		return 1
	}
	n := int64(bo.Uint16(t[ifd:]))
	for k := range n {
		p := ifd + 2 + k*dlugoscWpisuIFD
		if p+dlugoscWpisuIFD > int64(len(t)) {
			return 1
		}
		if bo.Uint16(t[p:]) != tagOrientacji {
			continue
		}
		if bo.Uint16(t[p+2:]) != typSHORT {
			return 1
		}
		if o := int(bo.Uint16(t[p+8:])); o >= 1 && o <= 8 {
			return o
		}
		return 1
	}
	return 1
}

// obroc stosuje orientację EXIF (2..8: odbicia i obroty), żeby miniatura stała prosto.
func obroc(src *image.RGBA, orientacja int) *image.RGBA {
	if orientacja < 2 || orientacja > 8 {
		return src
	}
	b := src.Bounds()
	sz, wys := b.Dx(), b.Dy()
	zamiana := orientacja >= 5 // 5..8: szerokość i wysokość się zamieniają
	dst := image.NewRGBA(image.Rect(0, 0, sz, wys))
	if zamiana {
		dst = image.NewRGBA(image.Rect(0, 0, wys, sz))
	}
	for y := range wys {
		for x := range sz {
			dx, dy := cel(orientacja, x, y, sz, wys)
			dst.SetRGBA(dx, dy, src.RGBAAt(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

// cel: gdzie trafia piksel (x, y) obrazu sz × wys przy danej orientacji.
func cel(orientacja, x, y, sz, wys int) (int, int) {
	switch orientacja {
	case 2: // odbicie poziome
		return sz - 1 - x, y
	case 3: // obrót 180
		return sz - 1 - x, wys - 1 - y
	case 4: // odbicie pionowe
		return x, wys - 1 - y
	case 5: // transpozycja
		return y, x
	case 6: // obrót 90 w prawo
		return wys - 1 - y, x
	case 7: // transwersja
		return wys - 1 - y, sz - 1 - x
	default: // 8: obrót 90 w lewo
		return y, sz - 1 - x
	}
}
