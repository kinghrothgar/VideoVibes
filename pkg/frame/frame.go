package frame

import (
	"image"
	"image/color"
	"log"
	"math"
	"sync"
)

func frameAvg(frame *image.RGBA) *color.RGBA {
	imgSize := frame.Bounds().Size()

	var redSum float64
	var greenSum float64
	var blueSum float64

	for x := 0; x < imgSize.X; x++ {
		for y := 0; y < imgSize.Y; y++ {
			pixel := frame.At(x, y)
			col := color.RGBAModel.Convert(pixel).(color.RGBA)

			redSum += float64(col.R)
			greenSum += float64(col.G)
			blueSum += float64(col.B)
		}
	}

	imgArea := float64(imgSize.X * imgSize.Y)

	red := uint8(math.Round(redSum / imgArea))
	green := uint8(math.Round(greenSum / imgArea))
	blue := uint8(math.Round(blueSum / imgArea))

	return &color.RGBA{red, green, blue, 0xff}
}

func HandleFrames(frames chan *image.RGBA, colors *[]color.RGBA, maxGoroutines int, done chan bool) chan bool {
	framesDone := make(chan bool)
	// TODO buffer length?
	colorsChan := make(chan *color.RGBA, maxGoroutines)
	// channel to close handleColors
	colorsClose := make(chan bool)
	colorsDone := handleColors(colors, colorsChan, colorsClose)
	go func() {
		var wg = sync.WaitGroup{}
		guard := make(chan struct{}, maxGoroutines)
	out:
		for {
			select {
			case frame := <-frames:
				guard <- struct{}{} // would block if guard channel is already filled
				wg.Add(1)
				go func() {
					colorsChan <- frameAvg(frame)
					<-guard
					wg.Done()
				}()
			case <-done:
				log.Println("HandleFrames notified done")
				for len(frames) > 0 {
					log.Println("handling last frames")
					guard <- struct{}{}
					frame := <-frames
					colorsChan <- frameAvg(frame)
				}
				colorsClose <- true
				break out
			}
		}
		log.Println("waiting on wg")
		wg.Wait()
		log.Println("waiting on colorsDone")
		<-colorsDone
		framesDone <- true
	}()
	return framesDone
}

func handleColors(colors *[]color.RGBA, colorsChan chan *color.RGBA, done chan bool) chan bool {
	colorsDone := make(chan bool)
	go func() {
	out:
		for {
			select {
			case color := <-colorsChan:
				*colors = append(*colors, *color)
			case <-done:
				log.Println("handleColors notified done")
				for len(colorsChan) > 0 {
					color := <-colorsChan
					*colors = append(*colors, *color)
				}
				break out
			}
		}
		colorsDone <- true
	}()
	return colorsDone
}
