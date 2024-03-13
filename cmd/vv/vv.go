package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"log"
	"math"
	"os"
	"runtime/pprof"
	"strconv"

	"github.com/kinghrothgar/VideoVibes/pkg/frame"
	"github.com/kinghrothgar/VideoVibes/pkg/media"
)

var (
	frameBufferSize = 2048
	maxGoroutines   = 128
	width           = 5120
	height          = 1440
	smoothing       = 1
	pngName         = "out.png"
)

func main() {
	f, err := os.Create("profile")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	m, err := media.NewMedia(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) >= 3 {
		width, err = strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
	}

	if len(os.Args) >= 4 {
		height, err = strconv.Atoi(os.Args[3])
		if err != nil {
			log.Fatal(err)
		}
	}
	if len(os.Args) >= 5 {
		smoothing, err = strconv.Atoi(os.Args[4])
		if err != nil {
			log.Fatal(err)
		}
	}

	if err := m.Open(); err != nil {
		log.Printf("failed to open decode media: %w", err)
	}
	defer m.Close()

	frames := make(chan *image.RGBA, frameBufferSize)
	echan := make(chan error, frameBufferSize)
	mediaDone := make(chan bool)
	frameColors := []color.RGBA{}
	go handleErrors(echan)
	framesDone := frame.HandleFrames(frames, &frameColors, maxGoroutines, mediaDone)
	mDone := m.Stream(frames, echan)
	<-mDone
	log.Println("media reading done")
	mediaDone <- true
	<-framesDone
	log.Println("frames proccessing done")

	fmt.Printf("%+v", frameColors)

	writeFrameData(frameColors)
	createImg(frameColors, width, height, smoothing)
}

func handleErrors(e chan error) {
	for {
		log.Println(<-e)
	}
}

func writeFrameData(frameColors []color.RGBA) {
	data, err := json.Marshal(frameColors)
	if err != nil {
		log.Println(err)
		return
	}
	f, _ := os.Create("frames.json")
	if _, err := f.Write(data); err != nil {
		log.Println(err)
	}
	f.Close()
}

func colorWindowAvg(frameColors []color.RGBA, window, smoothing, position int) color.RGBA {
	frameLen := len(frameColors)

	var redSum float64
	var greenSum float64
	var blueSum float64

	startFrame := window * position
	endFrame := min(startFrame+window*smoothing, frameLen)

	for i := window * position; i < endFrame; i++ {
		redSum += float64(frameColors[i].R)
		greenSum += float64(frameColors[i].G)
		blueSum += float64(frameColors[i].B)
	}

	frames := float64(endFrame - startFrame)
	red := uint8(math.Round(redSum / frames))
	green := uint8(math.Round(greenSum / frames))
	blue := uint8(math.Round(blueSum / frames))
	return color.RGBA{red, green, blue, 0xff}
}

func createImg(frameColors []color.RGBA, width, height, smoothing int) {
	window := len(frameColors) / width

	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})

	// Set color for each pixel.
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, colorWindowAvg(frameColors, window, smoothing, x))
		}
	}

	// Encode as PNG.
	f, _ := os.Create(pngName)
	png.Encode(f, img)
	f.Close()
}
