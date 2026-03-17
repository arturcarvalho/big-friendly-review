package main

import (
	"image"
	"image/color"
	"image/gif"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	scale    = 3
	padX     = 8
	padY     = 4
	charW    = 7 // basicfont.Face7x13 advance
	ascent   = 11
	descent  = 2
	flickerN = 5
)

var (
	bg      = color.RGBA{0xff, 0xff, 0xff, 0xff}
	fg      = color.RGBA{0x00, 0x00, 0x00, 0xff}
	pal     = color.Palette{bg, fg}
	rng     = rand.New(rand.NewSource(42))
	glyphs  = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*<>{}[]|/\\~")
)

type stage struct {
	text string
	hold int // centiseconds
}

func renderFrame(text string, w, h int) *image.Paletted {
	small := image.NewPaletted(image.Rect(0, 0, w, h), pal)

	textW := len(text) * charW
	x0 := (w - textW) / 2
	y0 := padY + ascent

	d := &font.Drawer{
		Dst:  small,
		Src:  image.NewUniform(fg),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x0, y0),
	}
	d.DrawString(text)

	bigW, bigH := w*scale, h*scale
	big := image.NewPaletted(image.Rect(0, 0, bigW, bigH), pal)
	for y := 0; y < bigH; y++ {
		for x := 0; x < bigW; x++ {
			big.SetColorIndex(x, y, small.ColorIndexAt(x/scale, y/scale))
		}
	}
	return big
}

func main() {
	stages := []stage{
		{"*******************", 0},
		{"Big ******** R***ew", 0},
		{"Big Fu***ing R*v*ew", 0},
		{"Big Friendly Review", 2000},
	}

	maxLen := 0
	for _, s := range stages {
		if len(s.text) > maxLen {
			maxLen = len(s.text)
		}
	}
	canvasW := maxLen*charW + padX*2
	canvasH := ascent + descent + padY*2

	g := &gif.GIF{LoopCount: 0}

	for i, s := range stages {
		if i > 0 {
			prev := stages[i-1].text
			for f := 0; f < flickerN; f++ {
				frame := make([]byte, len(s.text))
				for j := range s.text {
					if j < len(prev) && prev[j] == s.text[j] {
						frame[j] = s.text[j]
					} else if rng.Float64() < float64(f+1)/float64(flickerN+1) {
						frame[j] = s.text[j]
					} else {
						frame[j] = glyphs[rng.Intn(len(glyphs))]
					}
				}
				g.Image = append(g.Image, renderFrame(string(frame), canvasW, canvasH))
				g.Delay = append(g.Delay, 8)
			}
		}

		g.Image = append(g.Image, renderFrame(s.text, canvasW, canvasH))
		g.Delay = append(g.Delay, s.hold)
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot determine source path")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	outDir := filepath.Join(root, "assets")
	os.MkdirAll(outDir, 0o755)
	outPath := filepath.Join(root, "assets", "bfr.gif")

	f, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := gif.EncodeAll(f, g); err != nil {
		panic(err)
	}
	println("wrote", outPath)
}
