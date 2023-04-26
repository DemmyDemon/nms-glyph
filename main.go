package main

import (
	"bufio"
	"embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"io/fs"
	"net/http"

	"os"

	"github.com/go-chi/chi/v5"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

const (
	portalFont  = "res/NMS-Glyphs-Mono.ttf"
	fontDpi     = 72
	fontHinting = font.HintingNone
	fontSize    = 50
	cacheDir    = "cache"
	imgWidth    = 800
	imgHeight   = 45
	borderWidth = 1
)

var (
	//go:embed res
	res embed.FS
)

var (
	glyphColor    = color.RGBA{0x00, 0xB0, 0xBD, 0xFF}
	bgColor       = color.RGBA{0x00, 0x00, 0x00, 0x2C}
	bgColorImg    = image.NewUniform(bgColor)
	glyphColorImg = image.NewUniform(glyphColor)
)

// ReadFont reads font at the given path
func ReadFont(fontPath string) (*truetype.Font, error) {

	fontBytes, err := res.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("reading font: %w", err)
	}

	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing font: %w", err)
	}

	return f, nil

}

// createBlank creates the blank image to draw the glyphs on
func CreateBlank() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(img, img.Bounds(), bgColorImg, image.Point{}, draw.Src)

	if borderWidth <= 0 {
		return img
	}

	for x := 0; x < imgWidth; x++ {
		for y := 0; y < imgHeight; y++ {
			switch {
			case x < borderWidth:
				img.Set(x, y, glyphColor)
			case x >= imgWidth-borderWidth:
				img.Set(x, y, glyphColor)
			case y < borderWidth:
				img.Set(x, y, glyphColor)
			case y >= imgHeight-borderWidth:
				img.Set(x, y, glyphColor)
			}
		}
	}
	return img
}

// PrepareFreetypeContext sets up all the bits and bobs related to drawing text on the image
func PrepareFreetypeContext(dst *image.RGBA, font *truetype.Font) *freetype.Context {
	c := freetype.NewContext()
	c.SetDPI(fontDpi)
	c.SetFont(font)
	c.SetHinting(fontHinting)
	c.SetFontSize(fontSize)
	c.SetSrc(glyphColorImg)
	c.SetDst(dst)
	c.SetClip(dst.Bounds())

	return c
}

// DrawText draws the given text in the given context
func DrawText(c *freetype.Context, text string) error {
	baseline := (int(c.PointToFixed(fontSize) >> 6))
	pt := freetype.Pt(0, baseline-10)
	_, err := c.DrawString(text, pt)
	if err != nil {
		return fmt.Errorf("drawing text: %w", err)
	}
	return nil
}

// SaveToCache writes the given image to a file named after the given addresss
func SaveToCache(img *image.RGBA, address string) error {

	_, err := os.Stat(cacheDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = os.Mkdir(cacheDir, os.ModePerm)
			if err != nil {
				return fmt.Errorf("creating cache directory: %w", err)
			}
		} else {
			return fmt.Errorf("accessing cache directory: %w", err)
		}
	}

	outfile, err := os.Create(fmt.Sprintf("%s/%s.png", cacheDir, address))
	if err != nil {
		return fmt.Errorf("creating image file: %w", err)
	}
	defer outfile.Close()

	buf := bufio.NewWriter(outfile)
	err = png.Encode(buf, img)
	if err != nil {
		return fmt.Errorf("encoding image: %w", err)
	}
	err = buf.Flush()
	if err != nil {
		return fmt.Errorf("flushing image to disk: %w", err)
	}

	return nil
}

// CreatePortalImage creates an image with the given portal address on it
func CreatePortalImage(address string) (*image.RGBA, error) {
	font, err := ReadFont(portalFont)
	if err != nil {
		return nil, err
	}

	img := CreateBlank()

	c := PrepareFreetypeContext(img, font)

	err = DrawText(c, address)
	if err != nil {
		return nil, err
	}

	err = SaveToCache(img, address)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// ServeFromCache copies the file of the given filename to the given io.Writer
func ServeFromCache(w io.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cache open: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	return err
}

// WritePortalImage writes a portal address image of the given address to the given io.ResponseWriter.
// It also sets the Content-Type header to "img/png" in the process, and will generate the image if it
// does not exist in cache already.
func WritePortalImage(w http.ResponseWriter, address string) error {

	w.Header().Set("Content-Type", "image/png")

	filename := fmt.Sprintf("%s/%s.png", cacheDir, address)
	if _, err := os.Stat(filename); err == nil { // That is, the cache file exists and is all good!
		return ServeFromCache(w, filename)
	}

	// If we got here, then it was not cached, and we need to create it.
	img, err := CreatePortalImage(address)
	if err != nil {
		return fmt.Errorf("creating image: %w", err)
	}

	err = png.Encode(w, img) // FIXME: This does the encoding twice for uncached images!
	if err != nil {
		return fmt.Errorf("encoding to output: %w", err)
	}
	return nil
}

// RouteAddress simply gets the address from Chi and asks for a PNG showing that address.
func RouteAddress(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	err := WritePortalImage(w, address)
	if err != nil {
		fmt.Printf("ERROR encountered while trying to serve %s: %s\n", address, err)
	}
}

func main() {

	router := chi.NewRouter()
	router.Get("/{address:[0-9A-F]{16}}.png", RouteAddress)

	// Set SKIPEMBED var to nonzero to simplify development of the client-side stuff.
	// Otherwise, you'll need to recompile with every change to the HTML/CSS/JS...
	enableEmbed := os.Getenv("SKIPEMBED") == ""
	if enableEmbed {
		fmt.Printf("Using embedded files for web interface.")
		resHTML, err := fs.Sub(res, "res")
		if err != nil {
			panic(fmt.Sprintf("Unable to peer down embedded file tree: %s", err))
		}

		fileServer := http.FileServer(http.FS(resHTML))
		router.Handle("/*", fileServer)
	} else {
		fmt.Println("NOT using embedded files for web inteface")
		router.Handle("/*", http.FileServer(http.Dir("res")))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9192" // I couldn't find anything this might conflict with.
	}

	fmt.Printf("Listening on port %s\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), router)
	if err != nil {
		fmt.Printf("Shutting down: %s\n", err)
	}
}
