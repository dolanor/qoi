package qoi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
)

const Magic = "qoif"

type ChunkType byte

const (
	Index  ChunkType = 0x00
	Run8   ChunkType = 0x40
	Run16  ChunkType = 0x60
	Diff8  ChunkType = 0x80
	Diff16 ChunkType = 0xc0
	Diff24 ChunkType = 0xe0
	Color  ChunkType = 0xf0
)

type Mask byte

const (
	Mask2 Mask = 0xc0
	Mask3 Mask = 0xe0
	Mask4 Mask = 0xf0
)

type ColorSpace byte

const (
	ColorSpaceSRGB            = 0x00
	ColorSpaceSRGBLinearAlpha = 0x01
	ColorSpaceLinear          = 0x0f
)

const HeaderSize = 14
const Padding = 4

type header struct {
	Magic      [4]byte
	Width      uint32
	Height     uint32
	Channels   uint8
	ColorSpace uint8
}

func Decode(r io.Reader) (image.Image, error) {

	buf := bufio.NewReader(r)

	h := header{}
	err := binary.Read(buf, binary.BigEndian, &h)
	if err != nil {
		return nil, err
	}

	if string(h.Magic[:]) != Magic {
		return nil, fmt.Errorf("bad header magic value")
	}

	if h.Height == 0 || h.Width == 0 {
		return nil, fmt.Errorf("bad header height or width")
	}

	if h.Channels < 3 || h.Channels > 4 {
		return nil, fmt.Errorf("bad header channels")
	}

	img := image.NewNRGBA(image.Rect(0, 0, int(h.Width), int(h.Height)))
	run := 0

	pixels := img.Pix

	pix := color.RGBA{A: 255}
	const initPos = (0 ^ 0 ^ 0 ^ 255) % 64 // can't use colorHash here because of const
	seen := [64]color.RGBA{{}}
	_ = seen

	for len(pixels) > 0 {
		if run > 0 {
			run--
		} else {
			b, err := buf.ReadByte()
			if err == io.EOF {
				return img, nil
			}
			if err != nil {
				return nil, err
			}

			switch {
			case b&byte(Mask2) == byte(Index):
				pix = seen[b^byte(Index)]
			case b&byte(Mask3) == byte(Run8):
				run = int(b ^ 0x1f)
			case b&byte(Mask3) == byte(Run16):
				b2, err := buf.ReadByte()
				if err != nil {
					return nil, err
				}
				run = (int(b)^0x1f)<<8 | int(b2) + 32
			case b&byte(Mask2) == byte(Diff8):
				pix.R += ((b >> 4) & 0x03) - 2
				pix.G += ((b >> 2) & 0x03) - 2
				pix.B += (b & 0x03) - 2
			case b&byte(Mask3) == byte(Diff16):
				b2, err := buf.ReadByte()
				if err != nil {
					return nil, err
				}
				pix.R += (b & 0x1f) - 16
				pix.G += (b2 >> 4) - 8
				pix.B += (b2 & 0x0f) - 8
			case b&byte(Mask4) == byte(Diff24):
				b2, err := buf.ReadByte()
				if err != nil {
					return nil, err
				}
				b3, err := buf.ReadByte()
				if err != nil {
					return nil, err
				}
				pix.R += ((b&0x1f)<<1 | (b2 >> 7)) - 16
				pix.G += ((b2 & 0x7c) >> 2) - 16
				pix.B += ((b2&0x03)<<3 | ((b3 & 0xe0) >> 5)) - 16
				pix.A += (b3 & 0x1f) - 16
			case b&byte(Mask4) == byte(Color):
				switch {
				case b&8 != 0:
					b2, err := buf.ReadByte()
					if err != nil {
						return nil, err
					}
					pix.R = b2
				case b&4 != 0:
					b2, err := buf.ReadByte()
					if err != nil {
						return nil, err
					}
					pix.G = b2
				case b&2 != 0:
					b2, err := buf.ReadByte()
					if err != nil {
						return nil, err
					}
					pix.B = b2
				case b&1 != 0:
					b2, err := buf.ReadByte()
					if err != nil {
						return nil, err
					}
					pix.A = b2
				}
			default:
				pix = color.RGBA{}
			}
			seen[colorHash(pix)%64] = pix
		}

		n := copy(pixels[:4], []uint8{pix.R, pix.G, pix.B, pix.A})
		if n != 4 {
			return nil, errors.New("could not add pixel to image")
		}
		pixels = pixels[4:]
	}

	return img, nil
}

func Encode(w io.Writer, img image.Image) error {
	minX := uint32(img.Bounds().Min.X)
	maxX := uint32(img.Bounds().Max.X)
	minY := uint32(img.Bounds().Min.Y)
	maxY := uint32(img.Bounds().Max.Y)

	//buf := bufio.NewWriter(w)
	var buf bytes.Buffer

	// convert to static array
	m := (*[4]byte)([]byte(Magic))
	h := header{
		Magic:      *m,
		Width:      maxX - minX,
		Height:     maxY - minY,
		Channels:   4,              // FIXME: how do I get that from an image.Image?
		ColorSpace: ColorSpaceSRGB, // FIXME: how do I get that from an image.Image?
	}

	err := binary.Write(&buf, binary.BigEndian, h)
	if err != nil {
		return err
	}
	log.Println(minX, maxX, minY, maxY)
	log.Printf("heyho: %+v\n%+v\n", h, buf.Bytes())

	//img := image.NewNRGBA(image.Rect(0, 0, int(h.Width), int(h.Height)))
	run := 0

	pix := color.RGBA{A: 255}
	prev := pix
	const initPos = (0 ^ 0 ^ 0 ^ 255) % 64 // can't use colorHash here because of const
	seen := [64]color.RGBA{{}}
	_ = seen

	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			r, g, b, a := img.At(int(x), int(y)).RGBA()
			pix = color.RGBA{
				R: uint8(r),
				G: uint8(g),
				B: uint8(b),
				A: uint8(a),
			}
			if pix == prev {
				run++
			}

			lastPx := x == (maxX-1) && y == (maxY-1)
			if run > 0 && (run == 0x2020 || pix != prev || lastPx) {
				if run < 33 {
					run--
					err = binary.Write(&buf, binary.BigEndian, byte(int(Run8)|run))
					if err != nil {
						return fmt.Errorf("encode: run < 33: %w", err)
					}
				} else {
					run -= 33
					b := []byte{
						byte(int(Run16) | run>>8),
						byte(run),
					}
					err = binary.Write(&buf, binary.BigEndian, b)
					if err != nil {
						return fmt.Errorf("encode: run >= 33: %w", err)
					}

				}
				run = 0
			}

			if pix != prev {
				pos := colorHash(pix) % 64
				if seen[pos] == pix {
					err = binary.Write(&buf, binary.BigEndian, uint8(Index)|pos)
					if err != nil {
						return err
					}
				} else {

					seen[pos] = pix

					Δr := int(pix.R) - int(prev.R)
					Δg := int(pix.G) - int(prev.G)
					Δb := int(pix.B) - int(prev.B)
					Δa := int(pix.A) - int(prev.A)

					if true &&
						Δr > -17 && Δr < 16 &&
						Δg > -17 && Δg < 16 &&
						Δb > -17 && Δb < 16 &&
						Δa > -17 && Δa < 16 {

						if true &&
							Δa == 0 &&
							Δr > -3 && Δr < 2 &&
							Δg > -3 && Δg < 2 &&
							Δb > -3 && Δb < 2 {
							err = binary.Write(&buf, binary.BigEndian, uint8(int(Diff8)|((Δr+2)<<4|(Δg+2)<<2|(Δb+2))))
							if err != nil {
								return err
							}
						} else if true &&
							Δa == 0 &&
							Δr > -17 && Δr < 16 &&
							Δg > -9 && Δg < 8 &&
							Δb > -9 && Δb < 8 {
							b := []byte{
								byte(int(Diff16) | Δr + 16),
								byte((Δg+8)<<4 | Δb + 8),
							}

							err = binary.Write(&buf, binary.BigEndian, b)
							if err != nil {
								return err
							}
						} else {
							b := []byte{
								byte(int(Diff24) | Δr + 16>>1),
								byte((Δr+16)<<7 | (Δg+16)<<2 | (Δb+16)>>3),
								byte((Δb+16)<<5 | Δa + 16),
							}
							err = binary.Write(&buf, binary.BigEndian, b)
							if err != nil {
								return err
							}
						}

					} else {
						var r, g, b, a byte
						bs := []byte{0}
						log.Println("color changed:", Δr, Δg, Δb, Δa)
						if Δr != 0 {
							r = 8 // use bitmask type instead
							bs = append(bs, pix.R)
						}
						if Δg != 0 {
							g = 4
							bs = append(bs, pix.G)
						}
						if Δb != 0 {
							b = 2
							bs = append(bs, pix.B)
						}
						if Δa != 0 {
							a = 1
							bs = append(bs, pix.A)
						}
						bs[0] = byte(Color) | r | g | b | a

						err = binary.Write(&buf, binary.BigEndian, bs)
						if err != nil {
							return err
						}
					}

				}
			}
			prev = pix
		}
	}

	err = binary.Write(&buf, binary.BigEndian, [Padding]byte{})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, &buf)
	if err != nil {
		return err
	}

	return nil
}

func colorHash(c color.Color) uint8 {
	r, g, b, a := c.RGBA()
	return uint8(r ^ g ^ b ^ a)
}
