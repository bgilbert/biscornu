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
