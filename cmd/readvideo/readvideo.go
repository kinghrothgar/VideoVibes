package main

import (
	"image"
	"image/png"
	"os"

	"log"

	"github.com/kinghrothgar/VideoVibes/pkg/media"
)

const (
	frameBufferSize = 1024
)

func main() {
	// Open the media file.
	m, err := media.NewMedia(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	if err := m.Open(); err != nil {
		log.Printf("failed to open decode media: %w", err)
	}
	defer m.Close()

	frames := make(chan *image.RGBA, frameBufferSize)
	echan := make(chan error, frameBufferSize)
	go handleErrors(echan)
	go handleFrames(frames)
	mDone := m.Stream(frames, echan)
	<-mDone
}

func handleErrors(e chan error) {
	for {
		log.Println(<-e)
	}
}

func handleFrames(frames chan *image.RGBA) {
	for {
		<-frames
	}
}

func writeImg(name string, img *image.RGBA) {
	f, _ := os.Create(name)
	png.Encode(f, img)
	f.Close()
}
