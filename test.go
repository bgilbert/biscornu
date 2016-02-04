package main

import (
	"image"
)

func draw(mgr *PinManager, img *image.RGBA) {
	yStride := img.Rect.Dy() / 2
	for y := 0; y < yStride; y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			color := img.RGBAAt(x, y)
			mgr.Set(PIN_R1, color.R > 128)
			mgr.Set(PIN_G1, color.G > 128)
			mgr.Set(PIN_B1, color.B > 128)
			color = img.RGBAAt(x, y+yStride)
			mgr.Set(PIN_R2, color.R > 128)
			mgr.Set(PIN_G2, color.G > 128)
			mgr.Set(PIN_B2, color.B > 128)
			mgr.Strobe(PIN_CLK)
		}
		mgr.Set(PIN_OE, true)
		mgr.Strobe(PIN_LAT)
		mgr.Set(PIN_A3, y&0x8 != 0)
		mgr.Set(PIN_A2, y&0x4 != 0)
		mgr.Set(PIN_A1, y&0x2 != 0)
		mgr.Set(PIN_A0, y&0x1 != 0)
		mgr.Set(PIN_OE, false)
	}
}

func main() {
	// set up pin manager
	mgr, err := NewPinManager()
	if err != nil {
		panic(err)
	}
	defer mgr.Close()

	// enable everything but OE, clear the latches, then enable OE (active low)
	pins := make([]Pin, 0, 15)
	pins = append(pins, PINS_ADDRESS...)
	pins = append(pins, PINS_DATA...)
	pins = append(pins, PIN_CLK, PIN_LAT)
	for _, pin := range pins {
		if err := mgr.Add(pin); err != nil {
			panic(err)
		}
		mgr.Set(pin, false)
	}
	for i := 0; i < 32; i++ {
		mgr.Strobe(PIN_CLK)
	}
	mgr.Strobe(PIN_LAT)
	if err = mgr.Add(PIN_OE); err != nil {
		panic(err)
	}
	// make sure OE is removed first
	defer mgr.Remove(PIN_OE)
	mgr.Set(PIN_OE, false)

	// create image
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			color := img.RGBAAt(x, y)
			if x == y {
				color.R = 255
			}
			if x == 31-y {
				color.G = 255
			}
			if x == 16 {
				color.B = 255
			}
			color.A = 255
			img.SetRGBA(x, y, color)
		}
	}
	for {
		draw(mgr, img)
	}
}
