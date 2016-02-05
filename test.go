package main

import (
	"image"
	"os"
	"os/signal"
	"syscall"
)

const (
	WIDTH  = 32
	HEIGHT = 32
)

func paint(mgr *PinManager, img *image.RGBA) {
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

func painter(cimage <-chan image.RGBA, csig <-chan os.Signal, cdone chan<- bool) {
	// set exit action
	defer func() {
		cdone <- true
	}()

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
	for i := 0; i < WIDTH; i++ {
		mgr.Strobe(PIN_CLK)
	}
	mgr.Strobe(PIN_LAT)
	if err = mgr.Add(PIN_OE); err != nil {
		panic(err)
	}
	// make sure OE is removed first
	defer mgr.Remove(PIN_OE)
	mgr.Set(PIN_OE, false)

	// get initial image
	img := <-cimage

	// paint forever
	for {
		// check for signals, which have priority over new images
		select {
		case <-csig:
			return
		default:
		}

		// check for new image
		select {
		case img = <-cimage:
		default:
		}

		paint(mgr, &img)
	}
}

func main() {
	// exit on SIGINT/SIGTERM
	csig := make(chan os.Signal, 1)
	signal.Notify(csig, syscall.SIGINT, syscall.SIGTERM)

	// start painter
	cimage := make(chan image.RGBA)
	cdone := make(chan bool)
	go painter(cimage, csig, cdone)

	// create image and send it
	img := image.NewRGBA(image.Rect(0, 0, WIDTH, HEIGHT))
	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
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
	cimage <- *img

	// block until painter exits
	<-cdone
}
