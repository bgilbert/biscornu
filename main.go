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

	// start painter
	cimage := make(chan image.RGBA)
	cdone := make(chan bool)
	go display.Paint(cimage, csig, cdone)

	// send image
	cimage <- *img.(*image.RGBA)

	// block until painter exits
	<-cdone
}
