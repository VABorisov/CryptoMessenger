package ppm

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/VABorisov/CryptoMessenger/internal/image"
)

type PPMImage struct {
	logger        *slog.Logger
	width, height int
}

func NewPPMImage(logger *slog.Logger, width, height int) image.Image {
	return &PPMImage{
		logger: logger,
		width:  width,
		height: height,
	}
}

func (p *PPMImage) Encode(pixels []image.Pixel) ([]byte, error) {
	p.logger.Info("encoding PPM image")
	if len(pixels) != p.width*p.height {
		err := errors.New("pixel count does not match width*height")
		p.logger.Error("failed to encode PPM image", "error", err)
		return nil, fmt.Errorf("encoding PPM image error: %w", err)
	}

	buf := bytes.Buffer{}

	buf.WriteString(fmt.Sprintf("P6\n%d %d\n255\n", p.width, p.height))

	for _, px := range pixels {
		buf.WriteByte(px.R)
		buf.WriteByte(px.G)
		buf.WriteByte(px.B)
	}

	p.logger.Info("PPM image successfully encoded")

	return buf.Bytes(), nil
}

func (p *PPMImage) Decode(data []byte) ([]image.Pixel, error) {
	p.logger.Info("encoding PPM image")
	if len(data) < 3 {
		err := errors.New("data too short")
		p.logger.Error("failed to decode PPM image", "error", err)
		return nil, fmt.Errorf("decoding PPM image error: %w", err)
	}

	lines := strings.SplitN(string(data), "\n", 4)
	if len(lines) < 4 {
		err := errors.New("invalid header")
		p.logger.Error("failed to decode PPM image", "error", err)
		return nil, fmt.Errorf("decoding PPM image error: %w", err)
	}

	if lines[0] != "P6" {
		err := errors.New("not P6 format")
		p.logger.Error("failed to decode PPM image", "error", err)
		return nil, fmt.Errorf("decoding PPM image error: %w", err)
	}

	size := strings.Fields(lines[1])
	if len(size) != 2 {
		err := errors.New("invalid size")
		p.logger.Error("failed to decode PPM image", "error", err)
		return nil, fmt.Errorf("decoding PPM image error: %w", err)
	}
	width, _ := strconv.Atoi(size[0])
	height, _ := strconv.Atoi(size[1])
	p.width = width
	p.height = height

	headerLen := len(lines[0]) + len(lines[1]) + len(lines[2]) + 3
	raw := data[headerLen:]
	if len(raw) != p.width*p.height*3 {
		err := errors.New("pixel data length mismatch")
		p.logger.Error("failed to decode PPM image", "error", err)
		return nil, fmt.Errorf("decoding PPM image error: %w", err)
	}

	pixels := make([]image.Pixel, p.width*p.height)
	for i := 0; i < len(pixels); i++ {
		pixels[i] = image.Pixel{
			R: raw[i*3],
			G: raw[i*3+1],
			B: raw[i*3+2],
		}
	}

	return pixels, nil
}
