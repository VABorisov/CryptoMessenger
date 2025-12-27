package image

type Pixel struct {
	R, G, B uint8
}

type Image interface {
	Encode([]Pixel) ([]byte, error)
	Decode([]byte) ([]Pixel, error)
}
