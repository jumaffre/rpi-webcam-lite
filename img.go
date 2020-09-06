package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func formatImage(frame []byte, w uint32, h uint32) (buf *bytes.Buffer, err error) {
	// First, convert frame to YCbCr format
	yuyv := image.NewYCbCr(image.Rect(0, 0, int(w), int(h)), image.YCbCrSubsampleRatio422)
	for i := range yuyv.Cb {
		ii := i * 4
		yuyv.Y[i*2] = frame[ii]
		yuyv.Y[i*2+1] = frame[ii+2]
		yuyv.Cb[i] = frame[ii+1]
		yuyv.Cr[i] = frame[ii+3]
	}

	// Second, make image a RGBA and write current timestamp to it
	b := yuyv.Bounds()
	img := image.NewRGBA(b)
	draw.Draw(img, b, yuyv, b.Min, draw.Src)

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{200, 100, 0, 255}),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{fixed.Int26_6((img.Bounds().Max.X - 300) * 64), fixed.Int26_6((img.Bounds().Max.Y - 20) * 64)},
	}
	d.DrawString(time.Now().Format(time.RFC850))

	// Finally, jpeg encode image
	buf = &bytes.Buffer{}
	if err = jpeg.Encode(buf, img, nil); err != nil {
		log.Fatal(err)
		return
	}

	return buf, err
}
