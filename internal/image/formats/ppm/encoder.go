package ppm

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
)

var ErrUnsupportedColorMode = errors.New("ppm: color mode not supported")

func Encode(w io.Writer, img image.Image) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if _, err := fmt.Fprintf(bw, "P6\n%d %d\n255\n", width, height); err != nil {
		return err
	}

	switch img.ColorModel() {
	case color.RGBAModel:
		if rgba, ok := img.(*image.RGBA); ok {
			return encodeRGBA(bw, rgba)
		}
		return encodeRGBAImage(bw, img)
	default:
		return ErrUnsupportedColorMode
	}
}

func encodeRGBAImage(w io.Writer, img image.Image) error {
	bounds := img.Bounds()
	pixel := make([]byte, 3)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			pixel[0], pixel[1], pixel[2] = c.B, c.R, c.G
			if _, err := w.Write(pixel); err != nil {
				return err
			}
		}
	}
	return nil
}

func encodeRGBA(w io.Writer, img *image.RGBA) error {
	pix := img.Pix
	for i := 0; i < len(pix); i += 4 {
		pixel := []byte{pix[i+2], pix[i], pix[i+1]}
		if _, err := w.Write(pixel); err != nil {
			return err
		}
	}
	return nil
}
