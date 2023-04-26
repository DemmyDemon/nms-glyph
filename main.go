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

	"os"

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

func DrawText(c *freetype.Context, text string) error {
	baseline := (int(c.PointToFixed(fontSize) >> 6))
	pt := freetype.Pt(0, baseline-10)
	_, err := c.DrawString(text, pt)
	if err != nil {
		return fmt.Errorf("drawing text: %w", err)
	}
	return nil
}

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

func GetPortalImage(address string) (*image.RGBA, error) {

	// TODO: Check if it's in the cache, and just read it from there if it is

	img, err := CreatePortalImage(address)
	if err != nil {
		return nil, fmt.Errorf("creating image: %w", err)
	}
	return img, err
}

func main() {
	addreses := []string{
		"0123456789ABCDEF",
		"AA9239839217AFBB",
	}
	for _, address := range addreses {
		_, err := GetPortalImage(address)
		if err != nil {
			fmt.Printf("Failed to draw image for %s: %s\n", address, err)
			continue
		}
		fmt.Printf("Portal address %s now in cache!\n", address)
	}
}
