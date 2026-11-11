package main

import (
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"
)

// rgbaOf returns img as *image.RGBA without copying when it already has that pixel layout. d2d's frames are opaque
// (alpha 255), so an NRGBA from the PNG decoder has the same bytes as RGBA. The JPEG encoder only has a fast path for
// *image.RGBA (and YCbCr); every other type goes pixel by pixel through At, several times slower.
func rgbaOf(img image.Image) *image.RGBA {
	switch m := img.(type) {
	case *image.RGBA:
		return m
	case *image.NRGBA:
		if opaque(m.Pix) {
			return &image.RGBA{Pix: m.Pix, Stride: m.Stride, Rect: m.Rect}
		}
	}
	out := image.NewRGBA(img.Bounds())
	draw.Draw(out, out.Rect, img, img.Bounds().Min, draw.Src)
	return out
}

// opaque spot-checks the alpha bytes (every 97th pixel): a translucent picture takes the converting path.
func opaque(pix []byte) bool {
	for i := 3; i < len(pix); i += 4 * 97 {
		if pix[i] != 0xff {
			return false
		}
	}
	return true
}

// halve is a 2x2 box filter: a quarter of the pixels to encode and to send, for small viewports and phones.
func halve(src *image.RGBA) *image.RGBA {
	b := src.Rect
	w, h := b.Dx()/2, b.Dy()/2
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		r0 := src.Pix[(2*y)*src.Stride:]
		r1 := src.Pix[(2*y+1)*src.Stride:]
		d := dst.Pix[y*dst.Stride:]
		for x := 0; x < w; x++ {
			s := 8 * x
			for c := 0; c < 3; c++ {
				d[4*x+c] = uint8((uint(r0[s+c]) + uint(r0[s+4+c]) + uint(r1[s+c]) + uint(r1[s+4+c]) + 2) / 4)
			}
			d[4*x+3] = 0xff
		}
	}
	return dst
}

// encodeJPEG encodes one frame; buf is reused between calls.
func encodeJPEG(buf *bytes.Buffer, img *image.RGBA, half bool, quality int) ([]byte, error) {
	if half && img.Rect.Dx() >= 2 && img.Rect.Dy() >= 2 {
		img = halve(img)
	}
	buf.Reset()
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return bytes.Clone(buf.Bytes()), nil
}
