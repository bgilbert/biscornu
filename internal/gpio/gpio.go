package gpio

import (
	"errors"
	"fmt"
	"os"
	"runtime"
)

type Pin int

type Gpio struct {
	exporter   *os.File
	unexporter *os.File
	files      map[Pin]*os.File
	last       map[Pin]bool
}

func New() (mgr *Gpio, err error) {
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

	mgr = &Gpio{
		exporter:   exporter,
		unexporter: unexporter,
		files:      make(map[Pin]*os.File),
		last:       make(map[Pin]bool),
	}
	// ensure we unexport when garbage-collected
	runtime.SetFinalizer(mgr, (*Gpio).Close)
	return
}

func (mgr *Gpio) Add(pin Pin) (err error) {
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

func (mgr *Gpio) Remove(pin Pin) {
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

func (mgr *Gpio) Close() {
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

func (mgr *Gpio) Set(pin Pin, value bool) (err error) {
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

func (mgr *Gpio) Strobe(pin Pin) (err error) {
	err = mgr.Set(pin, true)
	if err == nil {
		err = mgr.Set(pin, false)
	}
	return
}
