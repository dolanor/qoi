package qoi_test

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"os"
	"testing"

	"github.com/dolanor/qoi"
)

func TestConst(t *testing.T) {
	//t.Fatalf("%x, %x", 0x40, 0x40+0x20)
	exp := []byte{
		0x00,
		0x40,
		0x60,
		0x80,
		0xc0,
		0xe0,
		0xf0,
	}
	for i, v := range []byte{
		byte(qoi.Index),
		byte(qoi.Run8),
		byte(qoi.Run16),
		byte(qoi.Diff8),
		byte(qoi.Diff16),
		byte(qoi.Diff24),
		byte(qoi.Color),
	} {
		t.Logf("%0X", v)
		if v != exp[i] {
			t.Errorf("\ngot: %x\nexp: %x", v, exp[i])
		}
	}

	f, err := os.Open("testdata/zero.qoi")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	qf, err := qoi.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	out, err := os.Create("testdata/image.png")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	err = png.Encode(out, qf)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFull(t *testing.T) {
	f, err := os.Open("testdata/small.test.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = qoi.Encode(&buf, img)
	if err != nil {
		t.Fatal(err)
	}
	qf, err := os.Create("testdata/tmp.qoi")
	if err != nil {
		t.Fatal(err)
	}
	defer qf.Close()

	_, err = io.Copy(qf, bytes.NewBuffer(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}

	qImg, err := qoi.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}

	out, err := os.Create("testdata/full.png")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	err = png.Encode(out, qImg)
	if err != nil {
		t.Fatal(err)
	}

}

func TestSmall(t *testing.T) {
	var pix []uint8
	var r, g, b uint8
	for i := 0; i < 30; i++ {
		ii := i / 10
		t.Log(ii)
		switch {
		case ii%3 == 0:
			t.Log("in", 3)
			b = 255
			g, r = 0, 0
		case ii%2 == 0:
			t.Log("in", 2)
			g = 255
			r, b = 0, 0
		case ii%1 == 0:
			t.Log("in", 1)
			r = 255
			g, b = 0, 0
		}
		pix = append(pix, r, g, b, 255)
	}
	t.Log("pix:", len(pix))
	img := image.NRGBA{
		Pix:    pix, //[]uint8{255, 0, 0, 255, 0, 0, 255, 255},
		Rect:   image.Rect(0, 0, 29, 2),
		Stride: 4,
	}
	out, err := os.Create("testdata/small.png")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	err = png.Encode(out, &img)
	if err != nil {
		t.Fatal(err)
	}
}
