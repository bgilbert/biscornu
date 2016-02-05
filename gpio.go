package main

import (
	"errors"
	"fmt"
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
	last       map[Pin]bool
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
		last:       make(map[Pin]bool),
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
	// drop state
	delete(mgr.last, pin)

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

	if last, ok := mgr.last[pin]; ok && value == last {
		return
	}
	defer func() {
		if err == nil {
			mgr.last[pin] = value
		}
	}()

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