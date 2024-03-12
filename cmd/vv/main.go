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
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

func mainB() {
	imgsPath := os.Args[1]
	maxGoroutines, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	outFrames, err := strconv.Atoi(os.Args[3])
	if err != nil {
		log.Fatal(err)
	}

	imgs, err := os.ReadDir(imgsPath)
	if err != nil {
		log.Fatal(err)
	}
	imgsLen := len(imgs)

	if imgsLen < outFrames {
		log.Fatal("Number of frame images less than outFrames")
	}

	frameColors := make([]color.RGBA, imgsLen)

	ticker := time.NewTicker(5 * time.Second)
	var wg = sync.WaitGroup{}
	guard := make(chan struct{}, maxGoroutines)
	for i, img := range imgs {
		guard <- struct{}{} // would block if guard channel is already filled
		select {
		case <-ticker.C:
			fmt.Printf("%s - %d%% (%d/%d)\n", time.Now().Format("01-02-2006 15:04:05"), i/imgsLen, i, imgsLen)
		default:
		}
		wg.Add(1)
		imgPath := filepath.Join(imgsPath, img.Name())
		go func(n int) {
			setAvgColor(&frameColors, n, imgPath)
			<-guard
			wg.Done()
		}(i)
	}

	wg.Wait()

	writeFrameData(frameColors)
	createImg(frameColors, outFrames, 1)
}

func main() {
	frameColorPath := os.Args[1]
	outFrames, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	smoothing, err := strconv.Atoi(os.Args[3])
	if err != nil {
		log.Fatal(err)
	}

	frameColorJSON, err := os.ReadFile(frameColorPath)
	if err != nil {
		log.Fatal(err)
	}
	var frameColors []color.RGBA
	err = json.Unmarshal(frameColorJSON, &frameColors)
	if err != nil {
		log.Fatal(err)
	}
	createImg(frameColors, outFrames, smoothing)
}

func setAvgColor(frameColors *[]color.RGBA, frame int, imgPath string) {
	file, err := os.Open(imgPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatalln(err)
	}

	imgSize := img.Bounds().Size()

	var redSum float64
	var greenSum float64
	var blueSum float64

	for x := 0; x < imgSize.X; x++ {
		for y := 0; y < imgSize.Y; y++ {
			pixel := img.At(x, y)
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

	(*frameColors)[frame] = color.RGBA{red, green, blue, 0xff}
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

func createImg(frameColors []color.RGBA, outFrames, smoothing int) {
	width := outFrames
	height := 500
	window := len(frameColors) / outFrames

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
	f, _ := os.Create("image_limited.png")
	png.Encode(f, img)
	f.Close()
}
