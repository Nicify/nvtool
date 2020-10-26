package main

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"syscall"
	"unsafe"

	g "github.com/AllenDang/giu"
	win "github.com/Nicify/nvtool/win"
	"github.com/fsnotify/fsnotify"
	"github.com/gobuffalo/packr/v2"
	"github.com/sqweek/dialog"
)

var (
	box = packr.New("assets", "./assets")
)

func nTrue(b ...bool) int {
	n := 0
	for _, v := range b {
		if v {
			n++
		}
	}
	return n
}

func byteCountDecimal(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

func byteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func loadImageFromMemory(imageData []byte) (imageRGBA *image.RGBA, err error) {
	r := bytes.NewReader(imageData)
	img, err := png.Decode(r)
	if err != nil {
		return nil, err
	}

	switch trueImg := img.(type) {
	case *image.RGBA:
		return trueImg, nil
	default:
		rgba := image.NewRGBA(trueImg.Bounds())
		draw.Draw(rgba, trueImg.Bounds(), trueImg, image.Pt(0, 0), draw.Src)
		return rgba, nil
	}
}

func imageToTexture(filename string) (*g.Texture, error) {
	imageByte, err := box.Find(filename)
	if err != nil {
		return nil, err
	}
	imageRGBA, _ := loadImageFromMemory(imageByte)
	textureID, err := g.NewTextureFromRgba(imageRGBA)
	return textureID, err
}

func selectInputPath() string {
	path, _ := dialog.File().Filter("video file", "mp4", "mkv", "mov", "flv", "avi").Load()
	return path
}

func selectOutputPath() string {
	path, _ := dialog.File().Filter("video file", "mp4", "mkv", "mov", "flv", "avi").Save()
	return path
}

func limitValue(val int32, min int32, max int32) int32 {
	if val > max {
		return max
	}
	if val < min {
		return min
	}
	return val
}

func limitResValue(val string) string {
	r := regexp.MustCompile(`[^0-9x]`)
	return r.ReplaceAllString(val, "")
}

func invalidPath(inputPath string, outputPath string) bool {
	return inputPath == outputPath || inputPath == "" || outputPath == ""
}

func getErrorLevel(processState *os.ProcessState) (int, bool) {
	if processState.Success() {
		return 0, true
	} else if t, ok := processState.Sys().(syscall.WaitStatus); ok {
		return t.ExitStatus(), true
	} else {
		return 255, false
	}
}

func initSingleInstanceLock() (unlock func()) {
	f, _ := os.Create(lockFile)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					command, _ := ioutil.ReadFile(lockFile)
					if string(command) == "focus" {
						win.ShowWindow(win.HWND(unsafe.Pointer(glfwWindow.GetWin32Window())), win.SW_RESTORE)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(lockFile)
	if err != nil {
		log.Fatal(err)
	}
	return func() {
		f.Close()
		watcher.Close()
	}
}
