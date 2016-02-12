package display

import (
	"errors"
	"github.com/bgilbert/biscornu/internal/gpio"
	"image"
)

// #include "interval.h"
import "C"

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

	fps       = 30
	colorBits = 2
	colors    = 1 << colorBits
	period    = 1000000000 / (fps * (Height / 2) * (colors - 1))
)

type Display struct {
	cimage chan image.RGBA
	cterm  chan bool
	cerr   chan error
}

func paint(mgr *gpio.Gpio, interval C.int, img *image.RGBA) {
	yStride := img.Rect.Dy() / 2
	for y := img.Rect.Min.Y; y < img.Rect.Min.Y+yStride; y++ {
		for i := 1; i < colors; i++ {
			thresh := uint8(i * 256 / colors)
			for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
				color := img.RGBAAt(x, y)
				mgr.Set(pinR1, color.R > thresh)
				mgr.Set(pinG1, color.G > thresh)
				mgr.Set(pinB1, color.B > thresh)
				color = img.RGBAAt(x, y+yStride)
				mgr.Set(pinR2, color.R > thresh)
				mgr.Set(pinG2, color.G > thresh)
				mgr.Set(pinB2, color.B > thresh)
				mgr.Strobe(pinClk)
			}
			// all set up to strobe; wait for end of interval
			C.interval_wait(interval)
			if i == 1 {
				mgr.Set(pinOE, true)
				mgr.Set(pinA3, y&0x8 != 0)
				mgr.Set(pinA2, y&0x4 != 0)
				mgr.Set(pinA1, y&0x2 != 0)
				mgr.Set(pinA0, y&0x1 != 0)
			}
			mgr.Strobe(pinLat)
			mgr.Set(pinOE, false)
		}
	}
}

func (disp *Display) paint() {
	// set up exit handler
	var err error
	defer func() {
		disp.cerr <- err
	}()

	// set up pin manager
	mgr, err := gpio.New()
	if err != nil {
		return
	}
	defer mgr.Close()

	// create interval
	interval := C.interval_create(period)
	if interval == -1 {
		err = errors.New("Couldn't create interval")
		return
	}
	defer C.interval_destroy(interval)

	// signal successful startup
	disp.cerr <- nil

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
	img := <-disp.cimage

	// paint forever
	for {
		// check for termination, which has priority over new images
		select {
		case <-disp.cterm:
			return
		default:
		}

		// check for new image
		select {
		case img = <-disp.cimage:
		default:
		}

		paint(mgr, interval, &img)
	}
}

func New() (disp *Display, err error) {
	disp = &Display{
		cimage: make(chan image.RGBA),
		cterm:  make(chan bool),
		cerr:   make(chan error),
	}
	go disp.paint()
	err = <-disp.cerr
	return
}

func (disp *Display) Frame(img *image.RGBA) {
	if img.Rect.Dx() != Width || img.Rect.Dy() != Height {
		panic("Incorrect image size")
	}
	disp.cimage <- *img
}

func (disp *Display) Stop() {
	disp.cterm <- true
	<-disp.cerr
}
