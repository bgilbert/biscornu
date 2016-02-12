package gpio

import (
	"errors"
	"runtime"
)

// #include "mmap.h"
import "C"

const (
	MMAP_BASE = 0x3f200000
	MMAP_SIZE = 160

	GPIO_OFF_FUNC  = 0x0
	GPIO_OFF_SET   = 0x1c
	GPIO_OFF_CLEAR = 0x28

	GPIO_FUNC_INPUT  = 0x0
	GPIO_FUNC_OUTPUT = 0x1

	MAX_PIN = 31
)

type Pin uint

type Gpio struct {
	hdl  *C.struct_range
	last map[Pin]bool
}

func New() (mgr *Gpio, err error) {
	hdl := C.range_map(MMAP_BASE, MMAP_SIZE)
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
	off := GPIO_OFF_FUNC + 4*(C.size_t(pin)/10)
	shift := 3 * (pin % 10)
	cur := uint(C.range_get_u32(mgr.hdl, off))
	val := (cur &^ (0x7 << shift)) | (mode << shift)
	C.range_set_u32(mgr.hdl, off, C.uint32_t(val))
}

func (mgr *Gpio) set(pin Pin, value bool) {
	var off C.size_t
	if value {
		off = GPIO_OFF_SET
	} else {
		off = GPIO_OFF_CLEAR
	}
	C.range_set_u32(mgr.hdl, off, 1<<pin)
	mgr.last[pin] = value
}

func (mgr *Gpio) Add(pin Pin) {
	if pin > MAX_PIN {
		panic("Requested unsupported pin")
	}
	mgr.set(pin, false)
	mgr.setFunc(pin, GPIO_FUNC_OUTPUT)
}

func (mgr *Gpio) Remove(pin Pin) {
	if _, ok := mgr.last[pin]; !ok {
		return
	}
	mgr.setFunc(pin, GPIO_FUNC_INPUT)
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
