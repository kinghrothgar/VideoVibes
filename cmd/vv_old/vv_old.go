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

var (
	maxGoroutines = 32
	width         = 5120
	height        = 1440
	smoothing     = 1
)

func main() {
	path := os.Args[1]

	var err error
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

	var frameColors []color.RGBA
	if isDir, err := IsDirectory(path); err != nil {
		log.Fatal(err)
	} else if isDir {
		frameColors = proccessPNGS(path)
	} else {
		frameColors = processFrameData(path)
	}
	if len(frameColors) < width {
		log.Fatal("Number of frame images less than width")
	}

	createImg(frameColors, width, height, smoothing)
}

func IsDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}

func proccessPNGS(imgsPath string) []color.RGBA {
	imgs, err := os.ReadDir(imgsPath)
	if err != nil {
		log.Fatal(err)
	}
	imgsLen := len(imgs)

	frameColors := make([]color.RGBA, imgsLen)

	ticker := time.NewTicker(5 * time.Second)
	var wg = sync.WaitGroup{}
	guard := make(chan struct{}, maxGoroutines)
	for i, img := range imgs {
		guard <- struct{}{} // would block if guard channel is already filled
		select {
		case <-ticker.C:
			percentage := int(math.Round(100 * float64(i) / float64(imgsLen)))
			fmt.Printf("%s - %d%% (%d/%d)\n", time.Now().Format("01-02-2006 15:04:05"), percentage, i, imgsLen)
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
	return frameColors
}

func processFrameData(frameColorPath string) []color.RGBA {
	frameColorJSON, err := os.ReadFile(frameColorPath)
	if err != nil {
		log.Fatal(err)
	}
	var frameColors []color.RGBA
	err = json.Unmarshal(frameColorJSON, &frameColors)
	if err != nil {
		log.Fatal(err)
	}
	return frameColors
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
	f, _ := os.Create("image_limited.png")
	png.Encode(f, img)
	f.Close()
}
