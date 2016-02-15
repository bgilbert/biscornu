/*
 * biscornu -  An odd little thing, covered with embroidery
 *
 * Copyright (C) 2016 Benjamin Gilbert
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of version 3 of the GNU General Public License as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

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
	hdl    *C.struct_range
	active map[Pin]bool
}

func New() (mgr *Gpio, err error) {
	hdl := C.range_map(mmapBase, mmapSize)
	if hdl == nil {
		err = errors.New("mmap failed")
		return
	}
	mgr = &Gpio{
		hdl:    hdl,
		active: make(map[Pin]bool),
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

func (mgr *Gpio) Add(pin Pin) {
	if pin > maxPin {
		panic("Requested unsupported pin")
	}
	mgr.Set(0, 1<<pin)
	mgr.setFunc(pin, gpioFuncOutput)
	mgr.active[pin] = true
}

func (mgr *Gpio) Remove(pin Pin) {
	if !mgr.active[pin] {
		return
	}
	mgr.setFunc(pin, gpioFuncInput)
	delete(mgr.active, pin)
}

func (mgr *Gpio) Close() {
	if mgr.hdl == nil {
		// already closed; perhaps we are running again as a finalizer
		return
	}
	for pin := range mgr.active {
		mgr.Remove(pin)
	}
	C.range_unmap(mgr.hdl)
	mgr.hdl = nil
}

func (mgr *Gpio) Set(states uint32, mask uint32) {
	val := states & mask
	if val != 0 {
		C.range_set_u32(mgr.hdl, gpioOffSet, C.uint32_t(val))
	}
	val = ^states & mask
	if val != 0 {
		C.range_set_u32(mgr.hdl, gpioOffClear, C.uint32_t(val))
	}
}

func (mgr *Gpio) Strobe(pin Pin) {
	mgr.Set(1<<pin, 1<<pin)
	mgr.Set(0, 1<<pin)
}
