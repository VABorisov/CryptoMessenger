package ppm

import (
	"bufio"
	"errors"
	"image"
	"image/color"
	"io"
	"strconv"
	"strings"
)

var (
	ErrBadHeader   = errors.New("ppm: invalid header")
	ErrNotEnough   = errors.New("ppm: not enough image data")
	ErrUnsupported = errors.New("ppm: unsupported format (maxVal != 255)")
)

func Decode(r io.Reader) (image.Image, error) {
	br := bufio.NewReader(r)

	magic, _ := br.ReadString('\n')
	magic = strings.TrimSpace(magic)
	if magic != "P6" {
		return nil, ErrBadHeader
	}

	readNonComment := func() string {
		for {
			line, _ := br.ReadString('\n')
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				return line
			}
		}
	}

	dims := strings.Split(readNonComment(), " ")
	if len(dims) != 2 {
		return nil, ErrBadHeader
	}
	width, err := strconv.Atoi(dims[0])
	if err != nil {
		return nil, ErrBadHeader
	}
	height, err := strconv.Atoi(dims[1])
	if err != nil {
		return nil, ErrBadHeader
	}

	maxVal, err := strconv.Atoi(readNonComment())
	if err != nil {
		return nil, ErrBadHeader
	}
	if maxVal != 255 {
		return nil, ErrUnsupported
	}

	_, _ = br.ReadByte()

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, _ := br.ReadByte()
			g, _ := br.ReadByte()
			b, _ := br.ReadByte()

			img.Set(x, y, color.RGBA{
				R: b,
				G: r,
				B: g,
				A: 255,
			})
		}
	}

	return img, nil
}
