package gpio

import (
	"errors"
	"runtime"
)

// #include "mmap.h"
import "C"

const (
	mmapBase = 0x3f200000
	mmapSize = 160

	gpioOffFunc  = 0x0
	gpioOffSet   = 0x1c
	gpioOffClear = 0x28

	gpioFuncInput  = 0x0
	gpioFuncOutput = 0x1

	maxPin = 31
)

type Pin uint

type Gpio struct {
	hdl  *C.struct_range
	last map[Pin]bool
}

func New() (mgr *Gpio, err error) {
	hdl := C.range_map(mmapBase, mmapSize)
	if hdl == nil {
		err = errors.New("mmap failed")
		return
	}
	mgr = &Gpio{
		hdl:  hdl,
		last: make(map[Pin]bool),
	}
	// ensure we unexport when garbage-collected
	runtime.SetFinalizer(mgr, (*Gpio).Close)
	return
}

func (mgr *Gpio) setFunc(pin Pin, mode uint) {
	off := gpioOffFunc + 4*(C.size_t(pin)/10)
	shift := 3 * (pin % 10)
	cur := uint(C.range_get_u32(mgr.hdl, off))
	val := (cur &^ (0x7 << shift)) | (mode << shift)
	C.range_set_u32(mgr.hdl, off, C.uint32_t(val))
}

func (mgr *Gpio) set(pin Pin, value bool) {
	var off C.size_t
	if value {
		off = gpioOffSet
	} else {
		off = gpioOffClear
	}
	C.range_set_u32(mgr.hdl, off, 1<<pin)
	mgr.last[pin] = value
}

func (mgr *Gpio) Add(pin Pin) {
	if pin > maxPin {
		panic("Requested unsupported pin")
	}
	mgr.set(pin, false)
	mgr.setFunc(pin, gpioFuncOutput)
}

func (mgr *Gpio) Remove(pin Pin) {
	if _, ok := mgr.last[pin]; !ok {
		return
	}
	mgr.setFunc(pin, gpioFuncInput)
	delete(mgr.last, pin)
}

func (mgr *Gpio) Close() {
	if mgr.hdl == nil {
		// already closed; perhaps we are running again as a finalizer
		return
	}
	for pin := range mgr.last {
		mgr.Remove(pin)
	}
	C.range_unmap(mgr.hdl)
	mgr.hdl = nil
}

func (mgr *Gpio) Set(pin Pin, value bool) {
	last, ok := mgr.last[pin]
	if !ok {
		panic("Attempted to set unconfigured pin")
	}
	if value != last {
		mgr.set(pin, value)
	}
}

func (mgr *Gpio) Strobe(pin Pin) {
	mgr.Set(pin, true)
	mgr.Set(pin, false)
}
