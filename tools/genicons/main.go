// Command genicons derives every icon asset WebVigil ships from the supplied masters in assets/.
// The masters are never modified; everything here is a scaled copy.
//
//	build/appicon.png                       512, used by Wails
//	build/windows/icon.ico                  multi-size PNG-in-ICO for WebVigil.exe
//	installer/build/appicon.png             512, the setup program's Wails icon
//	installer/build/windows/icon.ico        the same ICO on the setup program
//	installer/frontend/dist/icon.png        the setup header mark
//	installer/frontend/dist/light-mode.png  the setup theme toggle (target light)
//	installer/frontend/dist/dark-mode.png   the setup theme toggle (target dark)
//	frontend/src/assets/icons/<name>.png    one glyph per master, cropped to its ink
//	frontend/src/assets/donate.png          the donate mark, cropped and scaled by height
//	docs/icon.png                           256, the project site's icon
//
// Glyphs are cropped to their visible pixels and scaled to a common ink height, so a master drawn
// with a wide transparent margin renders at the same size as one drawn edge to edge.
//
// Run from the repo root: go run ./tools/genicons
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
)

const (
	mastersDir      = "assets"
	appMasterFile   = "application-icon.png"
	glyphOutputDir  = "frontend/src/assets/icons"
	installerDist   = "installer/frontend/dist"
	appIconSize     = 512
	installMarkSize = 252 // twice the 126px header mark, crisp at 200% scaling
	// The project site's icon: its favicon, header mark and Open Graph image.
	siteIconFile = "docs/icon.png"
	siteIconSize = 256
	// glyphHeight is four times --toolbar-glyph-size (61px, PigeonPost's title-bar glyph), so the
	// glyphs stay crisp under display scaling without carrying the 1254px masters into the binary.
	// Raising that CSS token means raising this with it; neither language can read the other's.
	glyphHeight = 244
	// glyphAlphaFloor is the 8-bit alpha at or below which a pixel counts as empty canvas when
	// cropping; the masters carry a faint glow far outside the drawing that a zero test reads as ink.
	glyphAlphaFloor = 8
	icoMaxDim       = 256
	icoBitsPerPixel = 32
	// The donate mark is a wide picture, not an icon: it is cropped to its artwork and scaled by
	// height alone, to four times the 29px the footer draws it at (PigeonPost's --titlebar-icon-size).
	donateMasterFile = "donate.png"
	donateHeight     = 116
	donateOutput     = "frontend/src/assets/donate.png"
)

var icoSizes = []int{16, 20, 24, 32, 40, 48, 64, 128, 256}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "genicons:", err)
		os.Exit(1)
	}
}

func run() error {
	master, err := loadPNG(filepath.Join(mastersDir, appMasterFile))
	if err != nil {
		return err
	}
	square := toSquare(master)
	appIcon := resize(square, appIconSize)
	for _, p := range []string{"build/appicon.png", "installer/build/appicon.png"} {
		if err := writePNG(p, appIcon); err != nil {
			return err
		}
	}
	frames := make([]*image.RGBA, 0, len(icoSizes))
	for _, size := range icoSizes {
		frames = append(frames, resize(square, size))
	}
	for _, p := range []string{"build/windows/icon.ico", "installer/build/windows/icon.ico"} {
		if err := writeICO(p, frames); err != nil {
			return err
		}
	}
	if err := writePNG(filepath.Join(installerDist, "icon.png"), resize(square, installMarkSize)); err != nil {
		return err
	}
	if err := writePNG(siteIconFile, resize(square, siteIconSize)); err != nil {
		return err
	}
	count, err := writeGlyphs()
	if err != nil {
		return err
	}
	donate, err := loadPNG(filepath.Join(mastersDir, donateMasterFile))
	if err != nil {
		return err
	}
	if err := writePNG(donateOutput, scaleToHeight(cropToArtwork(donate), donateHeight)); err != nil {
		return err
	}
	for _, name := range []string{"light-mode.png", "dark-mode.png"} {
		g, err := glyph(filepath.Join(mastersDir, name))
		if err != nil {
			return err
		}
		if err := writePNG(filepath.Join(installerDist, name), g); err != nil {
			return err
		}
	}
	fmt.Println("genicons: wrote app icons, both ICOs, the setup mark and", count, "glyphs")
	return nil
}

func writeGlyphs() (int, error) {
	entries, err := os.ReadDir(mastersDir)
	if err != nil {
		return 0, fmt.Errorf("read masters %q: %w", mastersDir, err)
	}
	written := 0
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".png") || strings.EqualFold(e.Name(), donateMasterFile) {
			continue
		}
		g, err := glyph(filepath.Join(mastersDir, e.Name()))
		if err != nil {
			return 0, err
		}
		if err := writePNG(filepath.Join(glyphOutputDir, strings.ToLower(e.Name())), g); err != nil {
			return 0, err
		}
		written++
	}
	return written, nil
}

func glyph(path string) (*image.RGBA, error) {
	m, err := loadPNG(path)
	if err != nil {
		return nil, err
	}
	return scaleToHeight(cropToArtwork(m), glyphHeight), nil
}

// cropToArtwork returns the tight box of visible pixels, keeping the aspect ratio.
func cropToArtwork(src image.Image) image.Image {
	const floor = glyphAlphaFloor * 0x101 // At() reports 16-bit alpha
	b := src.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := src.At(x, y).RGBA(); a <= floor {
				continue
			}
			minX, minY = min(minX, x), min(minY, y)
			maxX, maxY = max(maxX, x+1), max(maxY, y+1)
		}
	}
	if minX >= maxX || minY >= maxY {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, maxX-minX, maxY-minY))
	xdraw.Draw(dst, dst.Bounds(), src, image.Pt(minX, minY), xdraw.Src)
	return dst
}

func scaleToHeight(src image.Image, height int) *image.RGBA {
	b := src.Bounds()
	width := max(b.Dx()*height/b.Dy(), 1)
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst
}

func toSquare(src image.Image) *image.RGBA {
	b := src.Bounds()
	side := max(b.Dx(), b.Dy())
	dst := image.NewRGBA(image.Rect(0, 0, side, side))
	off := image.Pt((side-b.Dx())/2, (side-b.Dy())/2)
	xdraw.Draw(dst, image.Rectangle{Min: off, Max: off.Add(b.Size())}, src, b.Min, xdraw.Src)
	return dst
}

func resize(src image.Image, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open master %q: %w", path, err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode master %q: %w", path, err)
	}
	return img, nil
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir for %q: %w", path, err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("encode %q: %w", path, err)
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// writeICO writes a Vista-style ICO whose frames are PNG-compressed.
func writeICO(path string, frames []*image.RGBA) error {
	const headerSize, dirEntrySize = 6, 16
	blobs := make([][]byte, 0, len(frames))
	for _, f := range frames {
		var buf bytes.Buffer
		if err := png.Encode(&buf, f); err != nil {
			return fmt.Errorf("encode ico frame: %w", err)
		}
		blobs = append(blobs, buf.Bytes())
	}
	var out bytes.Buffer
	le := func(v any) { _ = binary.Write(&out, binary.LittleEndian, v) }
	le(uint16(0)) // reserved
	le(uint16(1)) // type: icon
	le(uint16(len(frames)))
	offset := headerSize + dirEntrySize*len(frames)
	for i, f := range frames {
		dim := byte(f.Bounds().Dx())
		if f.Bounds().Dx() >= icoMaxDim {
			dim = 0 // 0 means 256 in the ICO format
		}
		out.WriteByte(dim)
		out.WriteByte(dim)
		out.WriteByte(0) // palette size
		out.WriteByte(0) // reserved
		le(uint16(1))    // planes
		le(uint16(icoBitsPerPixel))
		le(uint32(len(blobs[i])))
		le(uint32(offset))
		offset += len(blobs[i])
	}
	for _, b := range blobs {
		out.Write(b)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir for %q: %w", path, err)
	}
	return os.WriteFile(path, out.Bytes(), 0o644)
}
