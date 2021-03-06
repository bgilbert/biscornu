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

package display

import (
	"encoding/binary"
	"errors"
	"github.com/bgilbert/biscornu/internal/gpio"
	"image"
	"io"
	"os"
	"reflect"
	"syscall"
)

// #include <sys/timerfd.h>
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
	cerr   chan error
}

func paint(mgr *gpio.Gpio, interval interval, img *image.RGBA) {
	yStride := img.Rect.Dy() / 2
	for y := img.Rect.Min.Y; y < img.Rect.Min.Y+yStride; y++ {
		for i := 1; i < colors; i++ {
			thresh := uint8(i * 256 / colors)
			for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
				var data uint32
				color := img.RGBAAt(x, y)
				if color.R >= thresh {
					data |= 1 << pinR1
				}
				if color.G >= thresh {
					data |= 1 << pinG1
				}
				if color.B >= thresh {
					data |= 1 << pinB1
				}
				color = img.RGBAAt(x, y+yStride)
				if color.R >= thresh {
					data |= 1 << pinR2
				}
				if color.G >= thresh {
					data |= 1 << pinG2
				}
				if color.B >= thresh {
					data |= 1 << pinB2
				}
				mgr.Set(data, 1<<pinR1|1<<pinG1|1<<pinB1|1<<pinR2|1<<pinG2|1<<pinB2)
				mgr.Strobe(pinClk)
			}
			// all set up to strobe; wait for end of interval
			interval.wait()
			if i == 1 {
				var addr uint32
				if y&0x8 != 0 {
					addr |= 1 << pinA3
				}
				if y&0x4 != 0 {
					addr |= 1 << pinA2
				}
				if y&0x2 != 0 {
					addr |= 1 << pinA1
				}
				if y&0x1 != 0 {
					addr |= 1 << pinA0
				}
				mgr.Set(1<<pinOE, 1<<pinOE)
				mgr.Set(addr, 1<<pinA3|1<<pinA2|1<<pinA1|1<<pinA0)
			}
			mgr.Strobe(pinLat)
			if i == 1 {
				mgr.Set(0, 1<<pinOE)
			}
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
	interval, err := newInterval(period)
	if err != nil {
		return
	}
	defer interval.close()

	// signal successful startup
	disp.cerr <- nil

	// enable everything but OE, clear the latches, then enable OE (active low)
	pins := make([]gpio.Pin, 0, 15)
	pins = append(pins, pinsAddress...)
	pins = append(pins, pinsData...)
	pins = append(pins, pinClk, pinLat)
	for _, pin := range pins {
		mgr.Add(pin)
	}
	for i := 0; i < Width; i++ {
		mgr.Strobe(pinClk)
	}
	mgr.Strobe(pinLat)
	mgr.Add(pinOE)
	// make sure OE is removed first
	defer mgr.Remove(pinOE)

	// get initial image
	img, ok := <-disp.cimage
	if !ok {
		return
	}

	// paint forever
	for {
		// check for new image or termination
		select {
		case img, ok = <-disp.cimage:
			if !ok {
				return
			}
		default:
		}

		paint(mgr, interval, &img)
	}
}

func New() (disp *Display, err error) {
	disp = &Display{
		cimage: make(chan image.RGBA),
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
	close(disp.cimage)
	<-disp.cerr
}

type interval struct {
	f io.ReadCloser
}

func newInterval(ns uint64) (_ interval, err error) {
	fd, _, errno := syscall.Syscall(syscall.SYS_TIMERFD_CREATE, C.CLOCK_MONOTONIC, C.TFD_CLOEXEC, 0)
	if fd == ^uintptr(0) {
		err = errors.New("Couldn't create timerfd: " + errno.Error())
		return
	}
	defer func() {
		if err != nil {
			syscall.Close(int(fd))
		}
	}()

	tspec := C.struct_timespec{
		tv_sec:  C.__time_t(ns / 1000000000),
		tv_nsec: C.__syscall_slong_t(ns % 1000000000),
	}
	ispec := C.struct_itimerspec{
		it_interval: tspec,
		it_value:    tspec,
	}
	ret, _, errno := syscall.Syscall6(syscall.SYS_TIMERFD_SETTIME, fd, 0, reflect.ValueOf(&ispec).Pointer(), 0, 0, 0)
	if ret == ^uintptr(0) {
		err = errors.New("Couldn't set interval: " + errno.Error())
		return
	}

	return interval{os.NewFile(fd, "timerfd")}, nil
}

func (interval interval) wait() (count uint64) {
	// We really want binary.NativeEndian, but Go doesn't have it
	err := binary.Read(interval.f, binary.LittleEndian, &count)
	if err != nil {
		return 0
	}
	return
}

func (interval interval) close() {
	interval.f.Close()
}
