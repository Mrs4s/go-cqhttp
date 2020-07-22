package global

import (
	"bytes"
	"fmt"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
)

// https://github.com/xrlin/AsciiArt

func Convert(f io.Reader, chars []string, subWidth, subHeight int, imageSwitch bool, bgColor, penColor color.RGBA) (string, *image.NRGBA, error) {
	var charsLength = len(chars)
	if charsLength == 0 {
		return "", nil, fmt.Errorf("no chars provided")
	}
	if subWidth == 0 || subHeight == 0 {
		return "", nil, fmt.Errorf("subWidth and subHeight params is required")
	}
	m, _, err := image.Decode(f)
	if err != nil {
		return "", nil, err
	}
	imageWidth, imageHeight := m.Bounds().Max.X, m.Bounds().Max.Y
	var img *image.NRGBA
	if imageSwitch {
		img = initImage(imageWidth, imageHeight, bgColor)
	}
	piecesX, piecesY := imageWidth/subWidth, imageHeight/subHeight
	var buff bytes.Buffer
	for y := 0; y < piecesY; y++ {
		offsetY := y * subHeight
		for x := 0; x < piecesX; x++ {
			offsetX := x * subWidth
			averageBrightness := calculateAverageBrightness(m, image.Rect(offsetX, offsetY, offsetX+subWidth, offsetY+subHeight))
			char := getCharByBrightness(chars, averageBrightness)
			buff.WriteString(char)
			if img != nil {
				addCharToImage(img, char, x*subWidth, y*subHeight, penColor)
			}
		}
		buff.WriteString("\n")
	}
	return buff.String(), img, nil
}

func initImage(width, height int, bgColor color.RGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, bgColor)
		}
	}
	return img
}
func calculateAverageBrightness(img image.Image, rect image.Rectangle) float64 {
	var averageBrightness float64
	width, height := rect.Max.X-rect.Min.X, rect.Max.Y-rect.Min.Y
	var brightness float64
	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			brightness = float64(r>>8+g>>8+b>>8) / 3
			averageBrightness += brightness
		}
	}
	averageBrightness /= float64(width * height)
	return averageBrightness
}

func getCharByBrightness(chars []string, brightness float64) string {
	index := int(brightness*float64(len(chars))) >> 8
	if index == len(chars) {
		index--
	}
	return chars[len(chars)-index-1]
}

func addCharToImage(img *image.NRGBA, char string, x, y int, penColor color.RGBA) {
	face := basicfont.Face7x13
	point := fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(penColor),
		Face: face,
		Dot:  point,
	}
	d.DrawString(char)
}

var Colors = map[string]color.RGBA{"black": {0, 0, 0, 255},
	"gray":  {140, 140, 140, 255},
	"red":   {255, 0, 0, 255},
	"green": {0, 128, 0, 255},
	"blue":  {0, 0, 255, 255}}
