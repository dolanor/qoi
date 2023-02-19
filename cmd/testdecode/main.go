package main

import (
	"image/png"
	"os"

	"github.com/dolanor/qoi"
)

var pixels []uint8

func main() {
	f, err := os.Open("../../testdata/zero.qoi")
	if err != nil {
		panic(err)
	}

	p, err := os.Open("../../testdata/zero.png")
	if err != nil {
		panic(err)
	}

	img, err := png.Decode(p)
	if err != nil {
		panic(err)
	}
	b := img.Bounds()
	x := b.Max.X - b.Min.X
	y := b.Max.Y - b.Min.Y
	pixels = make([]uint8, x*y*4)
	for j := 0; j < y; j++ {
		for i := 0; i < x; i++ {
			pix := img.At(i, j)
			r, g, b, a := pix.RGBA()
			pixels[i*j] = byte(r)
			pixels[i*j+1] = byte(g)
			pixels[i*j+2] = byte(b)
			pixels[i*j+3] = byte(a)
		}
	}

	qoiImg, err := qoi.Decode(f)
	if err != nil {
		panic(err)
	}
	_ = qoiImg

}
