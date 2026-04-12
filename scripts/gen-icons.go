package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math"
	"os"
	"path/filepath"
)

var (
	iconsDir = filepath.Join("apps", "desktop", "src-tauri", "icons")
	bg       = [4]byte{18, 18, 27, 255}
	accent   = [4]byte{249, 115, 22, 255}
	transp   = [4]byte{0, 0, 0, 0}
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	dir := filepath.Join(root, iconsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}

	fmt.Printf("Generating icons -> %s\n", dir)

	p32 := filepath.Join(dir, "32x32.png")
	p128 := filepath.Join(dir, "128x128.png")
	p256 := filepath.Join(dir, "128x128@2x.png")
	ico := filepath.Join(dir, "icon.ico")
	icns := filepath.Join(dir, "icon.icns")

	must(createRGBAIcon(32, 32, p32))
	must(createRGBAIcon(128, 128, p128))
	must(createRGBAIcon(256, 256, p256))
	must(createICO(p32, ico))
	must(createICNS(p128, icns))

	fmt.Println("Done. Replace these placeholders with production artwork before release.")
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func createRGBAIcon(width, height int, path string) error {
	var rows bytes.Buffer
	cx, cy := float64(width)/2, float64(height)/2
	rOuter := math.Min(float64(width), float64(height)) * 0.46
	rInner := math.Min(float64(width), float64(height)) * 0.42
	bolt := [][2]float64{
		{0.50, 0.05}, {0.28, 0.52}, {0.48, 0.52},
		{0.35, 0.95}, {0.72, 0.42}, {0.52, 0.42},
	}

	for y := 0; y < height; y++ {
		rows.WriteByte(0)
		for x := 0; x < width; x++ {
			dist := math.Hypot(float64(x)-cx, float64(y)-cy)
			switch {
			case dist > rOuter:
				rows.Write(transp[:])
			case dist > rInner:
				alpha := uint8(255 * (1 - (dist-rInner)/(rOuter-rInner)))
				rows.Write([]byte{accent[0], accent[1], accent[2], alpha})
			case pointInPolygon(float64(x), float64(y), bolt, width, height):
				rows.Write(accent[:])
			default:
				rows.Write(bg[:])
			}
		}
	}

	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(rows.Bytes()); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}

	var png bytes.Buffer
	png.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	png.Write(writePNGChunk("IHDR", mustBytes(func() ([]byte, error) {
		var b bytes.Buffer
		for _, v := range []uint32{uint32(width), uint32(height)} {
			if err := binary.Write(&b, binary.BigEndian, v); err != nil {
				return nil, err
			}
		}
		b.Write([]byte{8, 6, 0, 0, 0})
		return b.Bytes(), nil
	})))
	png.Write(writePNGChunk("IDAT", compressed.Bytes()))
	png.Write(writePNGChunk("IEND", nil))

	if err := os.WriteFile(path, png.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("  created %s (%dx%d)\n", filepath.Base(path), width, height)
	return nil
}

func createICO(pngPath, icoPath string) error {
	data, err := os.ReadFile(pngPath)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	if err := binary.Write(&out, binary.LittleEndian, uint16(0)); err != nil {
		return err
	}
	if err := binary.Write(&out, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(&out, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(&out, binary.LittleEndian, []byte{32, 32, 0, 0}); err != nil {
		return err
	}
	if err := binary.Write(&out, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(&out, binary.LittleEndian, uint16(32)); err != nil {
		return err
	}
	if err := binary.Write(&out, binary.LittleEndian, uint32(len(data))); err != nil {
		return err
	}
	if err := binary.Write(&out, binary.LittleEndian, uint32(22)); err != nil {
		return err
	}
	out.Write(data)
	return os.WriteFile(icoPath, out.Bytes(), 0o644)
}

func createICNS(pngPath, icnsPath string) error {
	data, err := os.ReadFile(pngPath)
	if err != nil {
		return err
	}
	chunkLen := uint32(8 + len(data))
	fileLen := uint32(8) + chunkLen

	var out bytes.Buffer
	out.WriteString("icns")
	if err := binary.Write(&out, binary.BigEndian, fileLen); err != nil {
		return err
	}
	out.WriteString("ic07")
	if err := binary.Write(&out, binary.BigEndian, chunkLen); err != nil {
		return err
	}
	out.Write(data)
	return os.WriteFile(icnsPath, out.Bytes(), 0o644)
}

func pointInPolygon(px, py float64, polygon [][2]float64, width, height int) bool {
	inside := false
	j := len(polygon) - 1
	for i := 0; i < len(polygon); i++ {
		xi, yi := polygon[i][0]*float64(width), polygon[i][1]*float64(height)
		xj, yj := polygon[j][0]*float64(width), polygon[j][1]*float64(height)
		if (yi > py) != (yj > py) && px < (xj-xi)*(py-yi)/(yj-yi)+xi {
			inside = !inside
		}
		j = i
	}
	return inside
}

func writePNGChunk(name string, data []byte) []byte {
	var out bytes.Buffer
	_ = binary.Write(&out, binary.BigEndian, uint32(len(data)))
	out.WriteString(name)
	out.Write(data)
	crc := crc32.ChecksumIEEE(append([]byte(name), data...))
	_ = binary.Write(&out, binary.BigEndian, crc)
	return out.Bytes()
}

func mustBytes(fn func() ([]byte, error)) []byte {
	b, err := fn()
	if err != nil {
		panic(err)
	}
	return b
}
