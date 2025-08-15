package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// --- Configuration ---
const (
	imgWidth  = 2456
	imgHeight = 1882
	outputDir = "elements"
	// !!! IMPORTANT: Change this path to a valid .ttf font file on your system !!!
	fontPath = "Roboto.ttf" // Example for Windows
	// fontPath = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf" // Example for Linux
)

// Element holds the chemical element data
type Element struct {
	Name         string  `json:"name"`
	Symbol       string  `json:"symbol"`
	AtomicNumber int     `json:"number"`
	AtomicMass   float64 `json:"atomic_mass"`
	Category     string  `json:"category"`
}

// --- Main Program ---

func main() {
	// 1. Load the font file
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		log.Fatalf("Failed to read font file: %v. Please check the 'fontPath' variable.", err)
	}
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Fatalf("Failed to parse font: %v", err)
	}

	// 2. Load and parse the colors.json file
	colorFile, err := os.ReadFile("colors.json")
	if err != nil {
		log.Fatalf("Failed to read colors.json: %v. Make sure the file exists in the same directory.", err)
	}
	var colorMap map[string]string
	if err := json.Unmarshal(colorFile, &colorMap); err != nil {
		log.Fatalf("Failed to parse colors.json: %v", err)
	}

	// 3. Unmarshal the embedded element JSON data
	var elements []Element
	var data struct {
		Elements []Element `json:"elements"`
	}
	if err := json.Unmarshal([]byte(periodicTableJSON), &data); err != nil {
		log.Fatalf("Failed to unmarshal element JSON data: %v", err)
	}
	elements = data.Elements

	// 4. Create the output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	fmt.Printf("Generating %d element images in '%s/' folder...\n", len(elements), outputDir)

	// 5. Loop through each element to create an image
	for _, el := range elements {
		img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

		// Get hex color from map, using "unknown" as a fallback
		hexStr, ok := colorMap[el.Category]
		if !ok {
			hexStr = colorMap["unknown"]
		}
		bgColor, err := parseHexColor(hexStr)
		if err != nil {
			log.Printf("Warning: Could not parse color '%s' for %s. Using gray.", hexStr, el.Name)
			bgColor = color.RGBA{R: 224, G: 224, B: 224, A: 255} // Default gray
		}
		draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

		c := freetype.NewContext()
		c.SetDPI(72)
		c.SetFont(f)
		c.SetClip(img.Bounds())
		c.SetDst(img)
		c.SetSrc(image.Black)

		// --- Draw Text Elements (Positions verified for correct layout) ---

		// Atomic Number (Top Left)
		c.SetFontSize(36)
		c.DrawString(fmt.Sprintf("%d", el.AtomicNumber), fixed.Point26_6{X: fixed.I(20), Y: fixed.I(50)})

		// Symbol (Center)
		c.SetFontSize(180)
		centerPointSymbol := fixed.Point26_6{X: fixed.I(imgWidth / 2), Y: fixed.I(imgHeight/2 + 30)}
		drawCenteredText(c, el.Symbol, centerPointSymbol, f, 180)

		// Name (Below Symbol)
		c.SetFontSize(40)
		yPosNameFloat := float64(imgHeight)*0.75 + 20
		yPosName := int(yPosNameFloat)
		centerPointName := fixed.Point26_6{X: fixed.I(imgWidth / 2), Y: fixed.I(yPosName)}
		drawCenteredText(c, el.Name, centerPointName, f, 40)

		// Atomic Mass (Bottom Center)
		c.SetFontSize(28)
		massStr := fmt.Sprintf("%.3f", el.AtomicMass)
		yPosMassFloat := float64(imgHeight)*0.88 + 20
		yPosMass := int(yPosMassFloat)
		centerPointMass := fixed.Point26_6{X: fixed.I(imgWidth / 2), Y: fixed.I(yPosMass)}
		drawCenteredText(c, massStr, centerPointMass, f, 28)

		// --- Save the image to a file ---
		filename := fmt.Sprintf("%03d-%s.png", el.AtomicNumber, el.Name)
		filepath := filepath.Join(outputDir, filename)
		outFile, err := os.Create(filepath)
		if err != nil {
			log.Printf("Failed to create file for %s: %v", el.Name, err)
			continue
		}
		defer outFile.Close()

		if err := png.Encode(outFile, img); err != nil {
			log.Printf("Failed to encode PNG for %s: %v", el.Name, err)
		}
		fmt.Println("Created", filepath)
	}

	fmt.Printf("\nDone! All %d element images have been saved.\n", len(elements))
}

// --- Helper Functions ---

// drawCenteredText measures a string and draws it so its center is at the given point.
func drawCenteredText(c *freetype.Context, text string, pt fixed.Point26_6, f *truetype.Font, size float64) {
	// Create a new face for measuring text
	face := truetype.NewFace(f, &truetype.Options{
		Size:    size,
		DPI:     72, // Use a standard DPI
		Hinting: font.HintingFull,
	})

	drawer := &font.Drawer{Face: face}
	width := drawer.MeasureString(text)
	pt.X -= width / 2

	// Draw the text
	_, err := c.DrawString(text, pt)
	if err != nil {
		log.Println(err)
	}
}

// parseHexColor converts a hex color string like "#FF0000" to a color.RGBA struct.
func parseHexColor(s string) (color.RGBA, error) {
	c := color.RGBA{A: 0xff}
	var err error
	if s[0] != '#' {
		return c, fmt.Errorf("invalid hex color format: missing #")
	}
	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		err = fmt.Errorf("invalid hex char: %c", b)
		return 0
	}
	switch len(s) {
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
	default:
		err = fmt.Errorf("invalid hex color length: %d", len(s))
	}
	return c, err
}

// Embedded JSON data for all 118 elements (for brevity)
const periodicTableJSON = `
{
  "elements": [
    {"name": "Hydrogen", "symbol": "H", "number": 1, "atomic_mass": 1.008, "category": "diatomic nonmetal"},
    {"name": "Helium", "symbol": "He", "number": 2, "atomic_mass": 4.002602, "category": "noble gas"},
    {"name": "Lithium", "symbol": "Li", "number": 3, "atomic_mass": 6.94, "category": "alkali metal"},
    {"name": "Beryllium", "symbol": "Be", "number": 4, "atomic_mass": 9.0121831, "category": "alkaline earth metal"},
    {"name": "Boron", "symbol": "B", "number": 5, "atomic_mass": 10.81, "category": "metalloid"},
    {"name": "Carbon", "symbol": "C", "number": 6, "atomic_mass": 12.011, "category": "polyatomic nonmetal"},
    {"name": "Nitrogen", "symbol": "N", "number": 7, "atomic_mass": 14.007, "category": "diatomic nonmetal"},
    {"name": "Oxygen", "symbol": "O", "number": 8, "atomic_mass": 15.999, "category": "diatomic nonmetal"},
    {"name": "Fluorine", "symbol": "F", "number": 9, "atomic_mass": 18.998403163, "category": "diatomic nonmetal"},
    {"name": "Neon", "symbol": "Ne", "number": 10, "atomic_mass": 20.1797, "category": "noble gas"},
    {"name": "Sodium", "symbol": "Na", "number": 11, "atomic_mass": 22.98976928, "category": "alkali metal"},
    {"name": "Magnesium", "symbol": "Mg", "number": 12, "atomic_mass": 24.305, "category": "alkaline earth metal"},
    {"name": "Aluminium", "symbol": "Al", "number": 13, "atomic_mass": 26.9815385, "category": "post-transition metal"},
    {"name": "Silicon", "symbol": "Si", "number": 14, "atomic_mass": 28.085, "category": "metalloid"},
    {"name": "Phosphorus", "symbol": "P", "number": 15, "atomic_mass": 30.973762, "category": "polyatomic nonmetal"},
    {"name": "Sulfur", "symbol": "S", "number": 16, "atomic_mass": 32.06, "category": "polyatomic nonmetal"},
    {"name": "Chlorine", "symbol": "Cl", "number": 17, "atomic_mass": 35.45, "category": "diatomic nonmetal"},
    {"name": "Argon", "symbol": "Ar", "number": 18, "atomic_mass": 39.948, "category": "noble gas"},
    {"name": "Potassium", "symbol": "K", "number": 19, "atomic_mass": 39.0983, "category": "alkali metal"},
    {"name": "Calcium", "symbol": "Ca", "number": 20, "atomic_mass": 40.078, "category": "alkaline earth metal"},
    {"name": "Scandium", "symbol": "Sc", "number": 21, "atomic_mass": 44.955908, "category": "transition metal"},
    {"name": "Titanium", "symbol": "Ti", "number": 22, "atomic_mass": 47.867, "category": "transition metal"},
    {"name": "Vanadium", "symbol": "V", "number": 23, "atomic_mass": 50.9415, "category": "transition metal"},
    {"name": "Chromium", "symbol": "Cr", "number": 24, "atomic_mass": 51.9961, "category": "transition metal"},
    {"name": "Manganese", "symbol": "Mn", "number": 25, "atomic_mass": 54.938044, "category": "transition metal"},
    {"name": "Iron", "symbol": "Fe", "number": 26, "atomic_mass": 55.845, "category": "transition metal"},
    {"name": "Cobalt", "symbol": "Co", "number": 27, "atomic_mass": 58.933194, "category": "transition metal"},
    {"name": "Nickel", "symbol": "Ni", "number": 28, "atomic_mass": 58.6934, "category": "transition metal"},
    {"name": "Copper", "symbol": "Cu", "number": 29, "atomic_mass": 63.546, "category": "transition metal"},
    {"name": "Zinc", "symbol": "Zn", "number": 30, "atomic_mass": 65.38, "category": "transition metal"},
    {"name": "Gallium", "symbol": "Ga", "number": 31, "atomic_mass": 69.723, "category": "post-transition metal"},
    {"name": "Germanium", "symbol": "Ge", "number": 32, "atomic_mass": 72.63, "category": "metalloid"},
    {"name": "Arsenic", "symbol": "As", "number": 33, "atomic_mass": 74.921595, "category": "metalloid"},
    {"name": "Selenium", "symbol": "Se", "number": 34, "atomic_mass": 78.971, "category": "polyatomic nonmetal"},
    {"name": "Bromine", "symbol": "Br", "number": 35, "atomic_mass": 79.904, "category": "diatomic nonmetal"},
    {"name": "Krypton", "symbol": "Kr", "number": 36, "atomic_mass": 83.798, "category": "noble gas"},
    {"name": "Rubidium", "symbol": "Rb", "number": 37, "atomic_mass": 85.4678, "category": "alkali metal"},
    {"name": "Strontium", "symbol": "Sr", "number": 38, "atomic_mass": 87.62, "category": "alkaline earth metal"},
    {"name": "Yttrium", "symbol": "Y", "number": 39, "atomic_mass": 88.90584, "category": "transition metal"},
    {"name": "Zirconium", "symbol": "Zr", "number": 40, "atomic_mass": 91.224, "category": "transition metal"},
    {"name": "Niobium", "symbol": "Nb", "number": 41, "atomic_mass": 92.90637, "category": "transition metal"},
    {"name": "Molybdenum", "symbol": "Mo", "number": 42, "atomic_mass": 95.95, "category": "transition metal"},
    {"name": "Technetium", "symbol": "Tc", "number": 43, "atomic_mass": 98, "category": "transition metal"},
    {"name": "Ruthenium", "symbol": "Ru", "number": 44, "atomic_mass": 101.07, "category": "transition metal"},
    {"name": "Rhodium", "symbol": "Rh", "number": 45, "atomic_mass": 102.9055, "category": "transition metal"},
    {"name": "Palladium", "symbol": "Pd", "number": 46, "atomic_mass": 106.42, "category": "transition metal"},
    {"name": "Silver", "symbol": "Ag", "number": 47, "atomic_mass": 107.8682, "category": "transition metal"},
    {"name": "Cadmium", "symbol": "Cd", "number": 48, "atomic_mass": 112.414, "category": "transition metal"},
    {"name": "Indium", "symbol": "In", "number": 49, "atomic_mass": 114.818, "category": "post-transition metal"},
    {"name": "Tin", "symbol": "Sn", "number": 50, "atomic_mass": 118.71, "category": "post-transition metal"},
    {"name": "Antimony", "symbol": "Sb", "number": 51, "atomic_mass": 121.76, "category": "metalloid"},
    {"name": "Tellurium", "symbol": "Te", "number": 52, "atomic_mass": 127.6, "category": "metalloid"},
    {"name": "Iodine", "symbol": "I", "number": 53, "atomic_mass": 126.90447, "category": "diatomic nonmetal"},
    {"name": "Xenon", "symbol": "Xe", "number": 54, "atomic_mass": 131.293, "category": "noble gas"},
    {"name": "Caesium", "symbol": "Cs", "number": 55, "atomic_mass": 132.90545196, "category": "alkali metal"},
    {"name": "Barium", "symbol": "Ba", "number": 56, "atomic_mass": 137.327, "category": "alkaline earth metal"},
    {"name": "Lanthanum", "symbol": "La", "number": 57, "atomic_mass": 138.90547, "category": "lanthanide"},
    {"name": "Cerium", "symbol": "Ce", "number": 58, "atomic_mass": 140.116, "category": "lanthanide"},
    {"name": "Praseodymium", "symbol": "Pr", "number": 59, "atomic_mass": 140.90766, "category": "lanthanide"},
    {"name": "Neodymium", "symbol": "Nd", "number": 60, "atomic_mass": 144.242, "category": "lanthanide"},
    {"name": "Promethium", "symbol": "Pm", "number": 61, "atomic_mass": 145, "category": "lanthanide"},
    {"name": "Samarium", "symbol": "Sm", "number": 62, "atomic_mass": 150.36, "category": "lanthanide"},
    {"name": "Europium", "symbol": "Eu", "number": 63, "atomic_mass": 151.964, "category": "lanthanide"},
    {"name": "Gadolinium", "symbol": "Gd", "number": 64, "atomic_mass": 157.25, "category": "lanthanide"},
    {"name": "Terbium", "symbol": "Tb", "number": 65, "atomic_mass": 158.92535, "category": "lanthanide"},
    {"name": "Dysprosium", "symbol": "Dy", "number": 66, "atomic_mass": 162.5, "category": "lanthanide"},
    {"name": "Holmium", "symbol": "Ho", "number": 67, "atomic_mass": 164.93033, "category": "lanthanide"},
    {"name": "Erbium", "symbol": "Er", "number": 68, "atomic_mass": 167.259, "category": "lanthanide"},
    {"name": "Thulium", "symbol": "Tm", "number": 69, "atomic_mass": 168.93422, "category": "lanthanide"},
    {"name": "Ytterbium", "symbol": "Yb", "number": 70, "atomic_mass": 173.045, "category": "lanthanide"},
    {"name": "Lutetium", "symbol": "Lu", "number": 71, "atomic_mass": 174.9668, "category": "lanthanide"},
    {"name": "Hafnium", "symbol": "Hf", "number": 72, "atomic_mass": 178.49, "category": "transition metal"},
    {"name": "Tantalum", "symbol": "Ta", "number": 73, "atomic_mass": 180.94788, "category": "transition metal"},
    {"name": "Tungsten", "symbol": "W", "number": 74, "atomic_mass": 183.84, "category": "transition metal"},
    {"name": "Rhenium", "symbol": "Re", "number": 75, "atomic_mass": 186.207, "category": "transition metal"},
    {"name": "Osmium", "symbol": "Os", "number": 76, "atomic_mass": 190.23, "category": "transition metal"},
    {"name": "Iridium", "symbol": "Ir", "number": 77, "atomic_mass": 192.217, "category": "transition metal"},
    {"name": "Platinum", "symbol": "Pt", "number": 78, "atomic_mass": 195.084, "category": "transition metal"},
    {"name": "Gold", "symbol": "Au", "number": 79, "atomic_mass": 196.966569, "category": "transition metal"},
    {"name": "Mercury", "symbol": "Hg", "number": 80, "atomic_mass": 200.592, "category": "transition metal"},
    {"name": "Thallium", "symbol": "Tl", "number": 81, "atomic_mass": 204.38, "category": "post-transition metal"},
    {"name": "Lead", "symbol": "Pb", "number": 82, "atomic_mass": 207.2, "category": "post-transition metal"},
    {"name": "Bismuth", "symbol": "Bi", "number": 83, "atomic_mass": 208.9804, "category": "post-transition metal"},
    {"name": "Polonium", "symbol": "Po", "number": 84, "atomic_mass": 209, "category": "post-transition metal"},
    {"name": "Astatine", "symbol": "At", "number": 85, "atomic_mass": 210, "category": "metalloid"},
    {"name": "Radon", "symbol": "Rn", "number": 86, "atomic_mass": 222, "category": "noble gas"},
    {"name": "Francium", "symbol": "Fr", "number": 87, "atomic_mass": 223, "category": "alkali metal"},
    {"name": "Radium", "symbol": "Ra", "number": 88, "atomic_mass": 226, "category": "alkaline earth metal"},
    {"name": "Actinium", "symbol": "Ac", "number": 89, "atomic_mass": 227, "category": "actinide"},
    {"name": "Thorium", "symbol": "Th", "number": 90, "atomic_mass": 232.0377, "category": "actinide"},
    {"name": "Protactinium", "symbol": "Pa", "number": 91, "atomic_mass": 231.03588, "category": "actinide"},
    {"name": "Uranium", "symbol": "U", "number": 92, "atomic_mass": 238.02891, "category": "actinide"},
    {"name": "Neptunium", "symbol": "Np", "number": 93, "atomic_mass": 237, "category": "actinide"},
    {"name": "Plutonium", "symbol": "Pu", "number": 94, "atomic_mass": 244, "category": "actinide"},
    {"name": "Americium", "symbol": "Am", "number": 95, "atomic_mass": 243, "category": "actinide"},
    {"name": "Curium", "symbol": "Cm", "number": 96, "atomic_mass": 247, "category": "actinide"},
    {"name": "Berkelium", "symbol": "Bk", "number": 97, "atomic_mass": 247, "category": "actinide"},
    {"name": "Californium", "symbol": "Cf", "number": 98, "atomic_mass": 251, "category": "actinide"},
    {"name": "Einsteinium", "symbol": "Es", "number": 99, "atomic_mass": 252, "category": "actinide"},
    {"name": "Fermium", "symbol": "Fm", "number": 100, "atomic_mass": 257, "category": "actinide"},
    {"name": "Mendelevium", "symbol": "Md", "number": 101, "atomic_mass": 258, "category": "actinide"},
    {"name": "Nobelium", "symbol": "No", "number": 102, "atomic_mass": 259, "category": "actinide"},
    {"name": "Lawrencium", "symbol": "Lr", "number": 103, "atomic_mass": 266, "category": "actinide"},
    {"name": "Rutherfordium", "symbol": "Rf", "number": 104, "atomic_mass": 267, "category": "transition metal"},
    {"name": "Dubnium", "symbol": "Db", "number": 105, "atomic_mass": 268, "category": "transition metal"},
    {"name": "Seaborgium", "symbol": "Sg", "number": 106, "atomic_mass": 269, "category": "transition metal"},
    {"name": "Bohrium", "symbol": "Bh", "number": 107, "atomic_mass": 270, "category": "transition metal"},
    {"name": "Hassium", "symbol": "Hs", "number": 108, "atomic_mass": 269, "category": "transition metal"},
    {"name": "Meitnerium", "symbol": "Mt", "number": 109, "atomic_mass": 278, "category": "unknown"},
    {"name": "Darmstadtium", "symbol": "Ds", "number": 110, "atomic_mass": 281, "category": "unknown"},
    {"name": "Roentgenium", "symbol": "Rg", "number": 111, "atomic_mass": 282, "category": "unknown"},
    {"name": "Copernicium", "symbol": "Cn", "number": 112, "atomic_mass": 285, "category": "transition metal"},
    {"name": "Nihonium", "symbol": "Nh", "number": 113, "atomic_mass": 286, "category": "unknown"},
    {"name": "Flerovium", "symbol": "Fl", "number": 114, "atomic_mass": 289, "category": "post-transition metal"},
    {"name": "Moscovium", "symbol": "Mc", "number": 115, "atomic_mass": 290, "category": "unknown"},
    {"name": "Livermorium", "symbol": "Lv", "number": 116, "atomic_mass": 293, "category": "unknown"},
    {"name": "Tennessine", "symbol": "Ts", "number": 117, "atomic_mass": 294, "category": "unknown"},
    {"name": "Oganesson", "symbol": "Og", "number": 118, "atomic_mass": 294, "category": "unknown"}
  ]
}
`

