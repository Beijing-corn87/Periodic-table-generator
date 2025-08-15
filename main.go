package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"log"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type SourceRoot struct {
	Elements []struct {
		Number     int     `json:"number"`
		Symbol     string  `json:"symbol"`
		Name       string  `json:"name"`
		AtomicMass float64 `json:"atomic_mass"`
		Category   string  `json:"category"`
		Xpos       int     `json:"xpos"`
		Ypos       int     `json:"ypos"`
	} `json:"elements"`
}

type Element struct {
	Number int
	Symbol string
	Name   string
	Mass   float64
	Type   string
}

type Colours map[string]string

func hexToRGBA(h string) color.RGBA {
	h = strings.TrimPrefix(strings.TrimSpace(h), "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return color.RGBA{0, 0, 0, 255}
	}
	var c color.RGBA
	fmt.Sscanf(h, "%02x%02x%02x", &c.R, &c.G, &c.B)
	c.A = 255
	return c
}

func normaliseCategory(c string) string {
	c = strings.ToLower(c)
	c = strings.ReplaceAll(c, "-", " ")
	c = strings.ReplaceAll(c, "_", " ")
	c = strings.Join(strings.Fields(c), " ")
	switch c {
	case "diatomic nonmetal", "polyatomic nonmetal":
		return "nonmetal"
	case "noble gas", "noble gases":
		return "noble gas"
	case "alkali metal", "alkali metals":
		return "alkali metal"
	case "alkaline earth metal", "alkaline earth metals":
		return "alkaline earth metal"
	case "transition metal", "transition metals":
		return "transition metal"
	case "post transition metal", "post transition metals":
		return "post-transition metal"
	case "lanthanide", "lanthanoid", "lanthanoids", "lanthanides":
		return "lanthanide"
	case "actinide", "actinoid", "actinoids", "actinides":
		return "actinide"
	default:
		return c
	}
}

func fetchElements() ([]Element, error) {
	client := http.Client{Timeout: 20 * time.Second}
	const url = "https://raw.githubusercontent.com/Bowserinator/Periodic-Table-JSON/master/PeriodicTableJSON.json"
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var root SourceRoot
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	var es []Element
	for _, e := range root.Elements {
		es = append(es, Element{
			Number: e.Number,
			Symbol: e.Symbol,
			Name:   e.Name,
			Mass:   e.AtomicMass,
			Type:   normaliseCategory(e.Category),
		})
	}
	sort.Slice(es, func(i, j int) bool { return es[i].Number < es[j].Number })
	if len(es) > 118 {
		es = es[:118]
	}
	return es, nil
}

func loadFont(path string, size float64) (font.Face, error) {
	fBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ft, err := opentype.Parse(fBytes)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

func drawText(img *image.RGBA, face font.Face, x, y int, txt string, col color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(txt)
}

func main() {
	// Settings
	const numSize float64 = 10
	const symSize float64 = 3
	const nameSize float64 = 7
	const massSize float64 = 10

	fontPath := flag.String("font", "font.ttf", "path to .ttf font file")
	coloursPath := flag.String("colours", "colours.json", "path to colours.json")
	outdir := flag.String("outdir", "elements", "output directory")
	height := flag.Int("height", 600, "tile image height in px (width scales to aspect ratio)")
	flag.Parse()

	// Load sample image to know aspect ratio
	file, err := os.Open("Untitled.png")
	if err != nil {
    	log.Fatal(err)
	}
	defer file.Close()

	ratio := float64(2456) / float64(1882)
	tileH := *height
	tileW := int(ratio * float64(tileH))

	// Read colours.json
	var colours Colours
	bs, err := os.ReadFile(*coloursPath)
	if err != nil {
		fmt.Println("Error reading colours.json:", err)
		return
	}
	if err := json.Unmarshal(bs, &colours); err != nil {
		fmt.Println("Error parsing colours.json:", err)
		return
	}

	// Load font faces of different sizes
	numFont, _ := loadFont(*fontPath, float64(tileH)/numSize)   // ~large enough
	symFont, _ := loadFont(*fontPath, float64(tileH)/symSize)    // biggest
	nameFont, _ := loadFont(*fontPath, float64(tileH)/nameSize)  // medium
	massFont, _ := loadFont(*fontPath, float64(tileH)/massSize)  // smallest

	// Fetch element data
	elements, err := fetchElements()
	if err != nil {
		fmt.Println("Error fetching elements:", err)
		return
	}

	os.MkdirAll(*outdir, 0755)

	for _, e := range elements {
		img := image.NewRGBA(image.Rect(0, 0, tileW, tileH))
		draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

		border := hexToRGBA(colours[e.Type])
		if _, ok := colours[e.Type]; !ok {
			border = color.RGBA{0, 0, 0, 255}
		}
		bt := tileH / 15 // border thickness proportional to height
		// Draw borders
		draw.Draw(img, image.Rect(0, 0, tileW, bt), &image.Uniform{border}, image.Point{}, draw.Src)
		draw.Draw(img, image.Rect(0, tileH-bt, tileW, tileH), &image.Uniform{border}, image.Point{}, draw.Src)
		draw.Draw(img, image.Rect(0, 0, bt, tileH), &image.Uniform{border}, image.Point{}, draw.Src)
		draw.Draw(img, image.Rect(tileW-bt, 0, tileW, tileH), &image.Uniform{border}, image.Point{}, draw.Src)

		// Padding
		pad := tileH / 20

		// Atomic Number (top-left)
		numTxt := fmt.Sprintf("%d", e.Number)
		drawText(img, numFont, bt+pad, bt+pad+int(numFont.Metrics().Height.Round()), numTxt, color.Black)

		// Atomic Mass (top-right)
		massTxt := fmt.Sprintf("%.4f", e.Mass)
		mw := font.MeasureString(massFont, massTxt).Round()
		drawText(img, massFont, tileW-bt-pad-mw, bt+pad+int(massFont.Metrics().Height.Round()), massTxt, color.Black)

		// Symbol (center)
		symW := font.MeasureString(symFont, e.Symbol).Round()
		drawText(img, symFont, (tileW-symW)/2, tileH/2, e.Symbol, color.Black)

		// Name (below symbol)
		nameW := font.MeasureString(nameFont, e.Name).Round()
		drawText(img, nameFont, (tileW-nameW)/2, tileH/2+int(nameFont.Metrics().Height.Round())+pad, e.Name, color.Black)

		// Save PNG
		fname := fmt.Sprintf("%03d_%s.png", e.Number, e.Symbol)
		f, _ := os.Create(filepath.Join(*outdir, fname))
		png.Encode(f, img)
		f.Close()
		fmt.Println("Written:", fname)
	}
}
