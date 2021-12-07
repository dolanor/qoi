package qoi_test

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/dolanor/qoi"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/image/bmp"
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
	err = qoi.Encode(&buf, img, nil)
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

var update = flag.Bool("update", false, "update golden files")

type DiffPanicWriter struct {
	pos    int
	golden []byte
	out    io.Writer
	t      *testing.T
}

func (w *DiffPanicWriter) Write(b []byte) (int, error) {
	w.t.Helper()
	for i := 0; i < len(b); i++ {
		if w.golden[w.pos] != b[i] {
			panic(fmt.Sprintf("tested data different than golden data: pos=%d: \n%x != %x", w.pos, w.golden[w.pos], b[i]))
		}
		w.pos++
		if len(b) < i-1 {
			return 0, errors.New("can't write 1 byte into writer")

		}
		n, err := w.out.Write([]byte{b[i]})
		if n != 1 && err != nil {
			return n, err
		}
	}

	return len(b), nil
}

func TestEncodeGolden(t *testing.T) {
	cases := map[string]struct {
		in      string
		golden  string
		options *qoi.Options
	}{
		"rgb 30x2": {"rgb.30x2.ori.png", "rgb.30x2.qoi", nil},
		"firefox":  {"firefox.ori.png", "firefox.ori.qoi", nil},
		"zero":     {"zero.ori.png", "zero.ori.qoi", &qoi.Options{Channels: 4, ColorSpace: qoi.ColorSpaceSRGBLinearAlpha}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			in, err := os.Open(filepath.Join("testdata", c.in))
			if err != nil {
				t.Fatal(err)
			}

			img, _, err := image.Decode(in)
			if err != nil {
				t.Fatal(err)
			}

			want, err := ioutil.ReadFile(filepath.Join("testdata", c.golden))
			if err != nil {
				t.Fatal(err)
			}

			var gott bytes.Buffer
			got := DiffPanicWriter{golden: []byte(want), out: &gott, t: t}
			err = qoi.Encode(&got, img, c.options)
			if err != nil {
				t.Fatal(err)
			}
			if *update {
				err = ioutil.WriteFile(filepath.Join("testdata", c.golden), gott.Bytes(), 0644)
				if err != nil {
					t.Fatal(err)
				}
				return
			}

			diff := cmp.Diff(want, gott.Bytes())
			if diff != "" {
				t.Errorf("encoded qoi file different from original qoi converted image (-want +got):\n%s", diff)
			}
		})
	}
}
