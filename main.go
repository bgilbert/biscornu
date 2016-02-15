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

package main

import (
	"github.com/bgilbert/biscornu/internal/display"
	"image"
	"image/png"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// exit on SIGINT/SIGTERM
	csig := make(chan os.Signal, 1)
	signal.Notify(csig, syscall.SIGINT, syscall.SIGTERM)

	// load image
	if len(os.Args) < 2 {
		panic("Specify input image")
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}

	// start display
	disp, err := display.New()
	if err != nil {
		panic(err)
	}

	// send image
	disp.Frame(img.(*image.RGBA))

	// wait for signal, stop display
	<-csig
	disp.Stop()
}
