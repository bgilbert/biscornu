package main

import (
	"github.com/bgilbert/biscornu/internal/gpio"
	"image"
	"image/color"
	"os"
	"os/signal"
	"syscall"
)

const (
	PIN_R1 = gpio.Pin(5)
	PIN_G1 = gpio.Pin(13)
	PIN_B1 = gpio.Pin(6)

	PIN_R2 = gpio.Pin(12)
	PIN_G2 = gpio.Pin(16)
	PIN_B2 = gpio.Pin(23)

	PIN_OE  = gpio.Pin(4)
	PIN_CLK = gpio.Pin(17)
	PIN_LAT = gpio.Pin(21)

	PIN_A0 = gpio.Pin(22)
	PIN_A1 = gpio.Pin(26)
	PIN_A2 = gpio.Pin(27)
	PIN_A3 = gpio.Pin(20)
)

var PINS_ADDRESS []gpio.Pin = []gpio.Pin{PIN_A0, PIN_A1, PIN_A2, PIN_A3}
var PINS_DATA []gpio.Pin = []gpio.Pin{PIN_R1, PIN_G1, PIN_B1, PIN_R2, PIN_G2, PIN_B2}

const (
	WIDTH  = 32
	HEIGHT = 32
)

func paint(mgr *gpio.Gpio, img *image.RGBA) {
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
	mgr, err := gpio.New()
	if err != nil {
		panic(err)
	}
	defer mgr.Close()

	// enable everything but OE, clear the latches, then enable OE (active low)
	pins := make([]gpio.Pin, 0, 15)
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

func imager(cimage chan<- image.RGBA) {
	// create images and send them
	img := image.NewRGBA(image.Rect(0, 0, WIDTH, HEIGHT))
	for frame := 0; ; frame++ {
		for y := 0; y < img.Rect.Dy(); y++ {
			for x := 0; x < img.Rect.Dx(); x++ {
				col := (x + frame) % img.Rect.Dx()
				cell := color.RGBA{
					A: 255,
				}
				if x == y {
					cell.R = 255
				}
				if x == 31-y {
					cell.G = 255
				}
				if x == 16 {
					cell.B = 255
				}
				img.SetRGBA(col, y, cell)
			}
		}
		cimage <- *img
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

	// start sending images
	go imager(cimage)

	// block until painter exits
	<-cdone
}
