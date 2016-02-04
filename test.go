package main

import (
	"errors"
	"fmt"
	"image"
	"os"
	"runtime"
)

type Pin int

const (
	PIN_R1 = Pin(5)
	PIN_G1 = Pin(13)
	PIN_B1 = Pin(6)

	PIN_R2 = Pin(12)
	PIN_G2 = Pin(16)
	PIN_B2 = Pin(23)

	PIN_OE  = Pin(4)
	PIN_CLK = Pin(17)
	PIN_LAT = Pin(21)

	PIN_A0 = Pin(22)
	PIN_A1 = Pin(26)
	PIN_A2 = Pin(27)
	PIN_A3 = Pin(20)
)

var PINS_ADDRESS []Pin = []Pin{PIN_A0, PIN_A1, PIN_A2, PIN_A3}
var PINS_DATA []Pin = []Pin{PIN_R1, PIN_G1, PIN_B1, PIN_R2, PIN_G2, PIN_B2}

type PinManager struct {
	exporter   *os.File
	unexporter *os.File
	files      map[Pin]*os.File
}

func NewPinManager() (mgr *PinManager, err error) {
	exporter, err := os.OpenFile("/sys/class/gpio/export", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			exporter.Close()
		}
	}()

	unexporter, err := os.OpenFile("/sys/class/gpio/unexport", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			unexporter.Close()
		}
	}()

	mgr = &PinManager{
		exporter:   exporter,
		unexporter: unexporter,
		files:      make(map[Pin]*os.File),
	}
	// ensure we unexport when garbage-collected
	runtime.SetFinalizer(mgr, func(mgr *PinManager) {
		mgr.Close()
	})
	return
}

func (mgr *PinManager) Add(pin Pin) (err error) {
	// check for double add
	if _, ok := mgr.files[pin]; ok {
		return errors.New("Pin already added")
	}

	// export pin
	if _, err = fmt.Fprintln(mgr.exporter, pin); err != nil {
		return
	}
	defer func() {
		if err != nil {
			mgr.Remove(pin)
		}
	}()
	if _, err = mgr.exporter.Seek(0, 0); err != nil {
		return
	}

	// set direction
	path := fmt.Sprintf("/sys/class/gpio/gpio%d/direction", pin)
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, "out")
	if err != nil {
		return
	}

	// open value file
	path = fmt.Sprintf("/sys/class/gpio/gpio%d/value", pin)
	if f, err = os.OpenFile(path, os.O_WRONLY, 0); err != nil {
		return
	}

	// commit
	mgr.files[pin] = f
	return
}

func (mgr *PinManager) Remove(pin Pin) {
	// close file
	if f, ok := mgr.files[pin]; ok {
		f.Close()
		delete(mgr.files, pin)
	}

	// try unexporting if exported; ignore errors
	path := fmt.Sprintf("/sys/class/gpio/gpio%d", pin)
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintln(mgr.unexporter, pin)
		mgr.unexporter.Seek(0, 0)
	}
}

func (mgr *PinManager) Close() {
	if mgr.files == nil {
		// already closed; perhaps we are running again as a finalizer
		return
	}
	for pin := range mgr.files {
		mgr.Remove(pin)
	}
	mgr.files = nil
	mgr.exporter.Close()
	mgr.unexporter.Close()
}

func (mgr *PinManager) Set(pin Pin, value bool) (err error) {
	f, ok := mgr.files[pin]
	if !ok {
		return errors.New("Pin not added")
	}
	var intValue int = 0
	if value {
		intValue = 1
	}
	_, err = fmt.Fprintln(f, intValue)
	_, err2 := f.Seek(0, 0)
	if err == nil && err2 != nil {
		err = err2
	}
	return
}

func (mgr *PinManager) Strobe(pin Pin) (err error) {
	err = mgr.Set(pin, true)
	if err == nil {
		err = mgr.Set(pin, false)
	}
	return
}

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
