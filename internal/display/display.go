package display

import (
	"github.com/bgilbert/biscornu/internal/gpio"
	"image"
	"os"
)

const (
	pinR1 = gpio.Pin(5)
	pinG1 = gpio.Pin(13)
	pinB1 = gpio.Pin(6)

	pinR2 = gpio.Pin(12)
	pinG2 = gpio.Pin(16)
	pinB2 = gpio.Pin(23)

	pinOE  = gpio.Pin(4)
	pinClk = gpio.Pin(17)
	pinLat = gpio.Pin(21)

	pinA0 = gpio.Pin(22)
	pinA1 = gpio.Pin(26)
	pinA2 = gpio.Pin(27)
	pinA3 = gpio.Pin(20)
)

var pinsAddress []gpio.Pin = []gpio.Pin{pinA0, pinA1, pinA2, pinA3}
var pinsData []gpio.Pin = []gpio.Pin{pinR1, pinG1, pinB1, pinR2, pinG2, pinB2}

const (
	Width  = 32
	Height = 32
)

func paint(mgr *gpio.Gpio, img *image.RGBA) {
	yStride := img.Rect.Dy() / 2
	for y := 0; y < yStride; y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			color := img.RGBAAt(x, y)
			mgr.Set(pinR1, color.R > 128)
			mgr.Set(pinG1, color.G > 128)
			mgr.Set(pinB1, color.B > 128)
			color = img.RGBAAt(x, y+yStride)
			mgr.Set(pinR2, color.R > 128)
			mgr.Set(pinG2, color.G > 128)
			mgr.Set(pinB2, color.B > 128)
			mgr.Strobe(pinClk)
		}
		mgr.Set(pinOE, true)
		mgr.Strobe(pinLat)
		mgr.Set(pinA3, y&0x8 != 0)
		mgr.Set(pinA2, y&0x4 != 0)
		mgr.Set(pinA1, y&0x2 != 0)
		mgr.Set(pinA0, y&0x1 != 0)
		mgr.Set(pinOE, false)
	}
}

func Paint(cimage <-chan image.RGBA, csig <-chan os.Signal, cdone chan<- bool) {
	// set exit action
	defer func() {
		cdone <- true
	}()

	// set up pin manager
	mgr, err := gpio.New()
	if err != nil {
		panic(err)
	}
	defer mgr.Close()

	// enable everything but OE, clear the latches, then enable OE (active low)
	pins := make([]gpio.Pin, 0, 15)
	pins = append(pins, pinsAddress...)
	pins = append(pins, pinsData...)
	pins = append(pins, pinClk, pinLat)
	for _, pin := range pins {
		mgr.Add(pin)
		mgr.Set(pin, false)
	}
	for i := 0; i < Width; i++ {
		mgr.Strobe(pinClk)
	}
	mgr.Strobe(pinLat)
	mgr.Add(pinOE)
	// make sure OE is removed first
	defer mgr.Remove(pinOE)
	mgr.Set(pinOE, false)

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
