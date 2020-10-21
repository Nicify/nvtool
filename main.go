package main

import (
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	g "github.com/AllenDang/giu"
	"github.com/AllenDang/giu/imgui"
	c "github.com/Nicify/nvtool/customwidget"
	mediainfo "github.com/Nicify/nvtool/mediainfo"
	nvenc "github.com/Nicify/nvtool/nvenc"
	theme "github.com/Nicify/nvtool/theme"
	win "github.com/Nicify/nvtool/win"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type encodingPresets struct {
	preset     int32
	rc         int32
	aq         int32
	cq         int32
	qmin       int32
	qmax       int32
	bitrate    int32
	maxrate    int32
	aqStrength int32
}

const (
	windowPadding = 8
	contentWidth  = 734
	buttonWidth   = 68
	buttonHeight  = 24
)

var (
	lockFile = path.Join(os.TempDir(), "nvtool.lock")
	nvencCmd *exec.Cmd

	fontTamzenr imgui.Font
	fontTamzenb imgui.Font
	fontIosevka imgui.Font

	texLogo         *g.Texture
	texButtonClose  *g.Texture
	texGraphicsCard *g.Texture

	mw         *g.MasterWindow
	glfwWindow *glfw.Window

	mwMoveable bool
	prevMouseX int
	prevMouseY int

	inputPath  string
	outputPath string
	percent    float32
	gpuName    string

	nvencLog     string
	mediaInfoLog string = "Drag and drop media files here"
)

var defaultPreset = encodingPresets{
	qmin:       16,
	qmax:       24,
	bitrate:    19850,
	maxrate:    59850,
	aqStrength: 15,
}

func isEncoding() bool {
	if nvencCmd == nil || (nvencCmd.ProcessState != nil && nvencCmd.ProcessState.Exited()) {
		return false
	}
	return true
}

func resetState() {
	percent = 0
	nvencLog = ""
	g.Update()
}

func onInputClick() {
	filePath := selectInputPath()
	if len(filePath) > 1 {
		percent = 0
		nvencLog = ""
		inputPath = filePath
		fileExt := path.Ext(inputPath)
		outputPath = strings.Replace(inputPath, fileExt, "_x264.mp4", 1)
		go setMediaInfo(filePath)
	}
}

func onOutputClick() {
	filePath := selectOutputPath()
	if len(filePath) > 1 {
		outputPath = filePath
	}
}

func onRunClick() {
	if isEncoding() ||
		invalidPath(inputPath, outputPath) ||
		strings.HasSuffix(nvencLog, "Get input file information...") {
		return
	}
	go func() {
		defer g.Update()
		resetState()
		command := fmt.Sprintf(
			"--profile high --audio-codec aac:aac_coder=twoloop --audio-bitrate 320 --preset P7 --vbr 0 --vbr-quality 12 --max-bitrate 60000 --lookahead 16 --strict-gop --aq-temporal --aq-strength 15 --mv-precision Q-pel --vpp-resize lanczos4 --vpp-perf-monitor --ssim --output-buf 128 --output-res 1280x720")
		cmd, progress, _ := nvenc.RunEncode(inputPath, outputPath, strings.Split(command, " "))
		nvencCmd = cmd
		for msg := range progress {
			percent = float32(msg.Percent) / 100
			nvencLog += fmt.Sprintf("%v frames: %.0f fps, %v kb/s, remain %s, est out size %s\n", msg.FramesProcessed, msg.FPS, msg.Bitrate, msg.Remain, msg.EstOutSize)
			g.Update()
		}
		// if nvencCmd.ProcessState != nil && nvencCmd.ProcessState.Success() {
		// 	nvencLog += "\nTranscoding success.\n"
		// 	return
		// }
		// nvencLog += "\nTranscoding failed.\n"
	}()
}

func setMediaInfo(inputPath string) {
	info, err := mediainfo.GetMediaInfo(inputPath)
	if err != nil {
		mediaInfoLog = fmt.Sprintf("Error: %s", err)
		return
	}
	mediaInfoLog = strings.Join(info, "\n")
	g.Update()
}

func onDrop(dropItem []string) {
	if isEncoding() {
		return
	}
	inputPath = dropItem[0]
	fileExt := path.Ext(inputPath)
	outputPath = strings.Replace(inputPath, fileExt, "_x264.mp4", 1)
	go setMediaInfo(inputPath)
}

func dispose() {
	if nvencCmd == nil {
		return
	}
	// nvencCmd.Process.Signal(os.Interrupt)
	if runtime.GOOS == "windows" {
		cmd := exec.Command("wmic", "process", "where", "name='NVEncC64.exe'", "delete")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Run()
	}
	go nvencCmd.Wait()
}

func shouldDisableInput(b bool) (flag g.WindowFlags) {
	if b {
		return g.WindowFlagsNoInputs
	}
	return
}

func shouldWindowMove() {
	mousePos := g.GetMousePos()
	prevPosX, prevPosY := glfwWindow.GetPos()
	if g.IsMouseClicked(0) {
		mwMoveable = float32(mousePos.Y) < 50*imgui.DPIScale
		prevMouseX = mousePos.X
		prevMouseY = mousePos.Y
	}
	if mwMoveable && g.IsMouseDown(0) {
		offsetX := mousePos.X - prevMouseX
		offsetY := mousePos.Y - prevMouseY
		glfwWindow.SetPos(prevPosX+int(offsetX), prevPosY+int(offsetY))
	}
}

func loop() {
	shouldWindowMove()
	isEncoding := isEncoding()
	useLayoutFlat := theme.UseLayoutFlat()
	useStyleDarkButton := theme.UseStyleDarkButton()
	defer useLayoutFlat.Pop()
	useLayoutFlat.Push()
	g.SingleWindow("NVTool",
		g.Layout{
			g.Line(
				g.Image(texLogo, 18, 18),
				g.Label("NVENC Video Toolbox 1.5"),
				g.Dummy(-83, 0),
				g.Custom(useStyleDarkButton.Push),
				g.ButtonV(".", 20, 20, func() {}),
				g.ButtonV("_", 20, 20, func() {
					win.ShowWindow(win.HWND(unsafe.Pointer(glfwWindow.GetWin32Window())), win.SW_FORCEMINIMIZE)
				}),
				g.ImageButton(texButtonClose, 20, 20, func() {
					glfwWindow.SetShouldClose(true)
				}),
				g.Custom(useStyleDarkButton.Pop),
			),
			g.TabBar("maintab", g.Layout{
				g.TabItem("Encode", g.Layout{
					g.Child("control", false, contentWidth, 92, shouldDisableInput(isEncoding), g.Layout{
						g.Spacing(),
						g.Line(
							g.InputTextV("##video", -((windowPadding+buttonWidth)/imgui.DPIScale), &inputPath, 0, nil, nil),
							c.WithHiDPIFont(fontIosevka, fontTamzenb, g.Layout{g.ButtonV("Video", buttonWidth, buttonHeight, onInputClick)}),
						),

						g.Spacing(),
						g.Line(
							g.InputTextV("##output", -((windowPadding+buttonWidth)/imgui.DPIScale), &outputPath, 0, nil, nil),
							c.WithHiDPIFont(fontIosevka, fontTamzenb, g.Layout{g.ButtonV("Output", buttonWidth, buttonHeight, onOutputClick)}),
						),

						g.Spacing(),
						g.Line(
							g.Label("Preset"),
							g.Combo("##preset", nvenc.PresetOptions[defaultPreset.preset], nvenc.PresetOptions, &defaultPreset.preset, 80, 0, nil),

							// g.Label("RC"),
							// g.Combo("##rc", nvenc.RCOptions[defaultPreset.rc], nvenc.RCOptions, &defaultPreset.rc, 80, 0, nil),

							g.Label("QMin"),
							g.InputIntV("##qmin", 40, &defaultPreset.qmin, 0, nil),

							g.Label("QMax"),
							g.InputIntV("##qmax", 40, &defaultPreset.qmax, 0, nil),

							g.Label("AQ"),
							g.Combo("##aq", nvenc.AQOptions[defaultPreset.aq], nvenc.AQOptions, &defaultPreset.aq, 95, 0, nil),

							g.InputIntV("##aqstrength", 40, &defaultPreset.aqStrength, 0, func() {
								defaultPreset.aqStrength = limitValue(defaultPreset.aqStrength, 0, 15)
							}),

							g.Label("Bitrate"),
							g.InputIntV("##bitrate", 70, &defaultPreset.bitrate, 0, nil),

							// g.Label("Maxrate"),
							// g.InputIntV("k##maxrate", 65, &defaultPreset.maxrate, 0, nil),
						),
					}),

					g.Spacing(),
					g.InputTextMultiline("##nvencLog", &nvencLog, contentWidth, 200, g.InputTextFlagsReadOnly, nil, func() {
						imgui.SetScrollHereY(1.0)
					}),

					g.Spacing(),
					g.ProgressBar(percent, contentWidth, 20, ""),

					g.Line(
						g.Dummy(0, 5),
					),
					g.Line(
						g.Condition(gpuName != "", g.Layout{
							g.Line(
								g.Image(texGraphicsCard, 18, 18),
								g.Label(gpuName),
							),
						}, nil),
						g.Dummy(-(windowPadding+buttonWidth), 24),
						c.WithHiDPIFont(fontIosevka, fontTamzenb, g.Layout{g.Condition(isEncoding,
							g.Layout{g.ButtonV("Cancel", buttonWidth, buttonHeight, dispose)},
							g.Layout{g.ButtonV("Run", buttonWidth, buttonHeight, onRunClick)},
						)}),
					),
				},
				),

				g.TabItem("MediaInfo", g.Layout{
					g.Spacing(),
					g.InputTextMultiline("##mediaInfoLog", &mediaInfoLog, contentWidth, 360, g.InputTextFlagsReadOnly, nil, nil),
				}),

				// g.TabItem("Settings", g.Layout{
				// 	g.Custom(func() {
				// 		imgui.PushStyleColor(imgui.StyleColorChildBg, imgui.Vec4{X: 0.12, Y: 0.12, Z: 0.12, W: 0.99})
				// 	}),

				// 	g.Spacing(),
				// 	g.Label("Interface"),
				// 	g.Child("Interface", true, contentWidth, 95, g.WindowFlagsAlwaysUseWindowPadding, g.Layout{}),

				// 	g.Spacing(),
				// 	g.Label("Encoding"),
				// 	g.Child("Encoding", true, contentWidth, 95, g.WindowFlagsAlwaysUseWindowPadding, g.Layout{}),

				// 	g.Spacing(),
				// 	g.Label("Binary"),
				// 	g.Child("Binary", true, contentWidth, 95, g.WindowFlagsAlwaysUseWindowPadding, g.Layout{}),

				// 	g.Custom(func() {
				// 		imgui.PopStyleColorV(1)
				// 	}),
				// }),
			}),
		})
}

func applyWindowProperties(window *glfw.Window) {
	data, err := box.Find("icon_48px.png")
	if err != nil {
		log.Fatal("icon_48px.png read failed.")
	}
	icon48px, _ := loadImageFromMemory(data)
	glfwWindow.SetIcon([]image.Image{icon48px})
	hwnd := win.HWND(unsafe.Pointer(glfwWindow.GetWin32Window()))
	win.SetWindowCompositionAttribute(hwnd, 3, 0, 0, 0)
	glfwWindow.SetFocusCallback(func(w *glfw.Window, focused bool) {
		if focused {
			glfwWindow.SetOpacity(0.98)
			return
		}
		glfwWindow.SetOpacity(1)
	})
}

func loadFont() {
	fonts := g.Context.IO().Fonts()
	fontIosevkaTTF, _ := box.Find("iosevka.ttf")
	fontIosevka = fonts.AddFontFromMemoryTTFV(fontIosevkaTTF, 18, imgui.DefaultFontConfig, fonts.GlyphRangesChineseFull())
	fontTamzenbTTF, _ := box.Find("tamzen8x16b.ttf")
	fontTamzenb = fonts.AddFontFromMemoryTTFV(fontTamzenbTTF, 16, imgui.DefaultFontConfig, fonts.GlyphRangesChineseFull())
	fontTamzenrTTF, _ := box.Find("tamzen8x16r.ttf")
	fontTamzenr = fonts.AddFontFromMemoryTTFV(fontTamzenrTTF, 16, imgui.DefaultFontConfig, fonts.GlyphRangesChineseFull())
}

func loadTexture() {
	texLogo, _ = imageToTexture("icon.png")
	texButtonClose, _ = imageToTexture("close_white.png")
	texGraphicsCard, _ = imageToTexture("graphics_card.png")
}

func init() {
	runtime.LockOSThread()

	if err := os.Remove(lockFile); err != nil && !os.IsNotExist(err) {
		ioutil.WriteFile(lockFile, []byte("focus"), 0644)
		os.Exit(0)
	}
}

func main() {
	defer dispose()
	unlock := initSingleInstanceLock()
	defer unlock()
	go loadTexture()
	mw = g.NewMasterWindow("NVTool", 750, 435, g.MasterWindowFlagsNotResizable|g.MasterWindowFlagsFrameless|g.MasterWindowFlagsTransparent, loadFont)
	currentStyle := imgui.CurrentStyle()
	theme.SetThemeDark(&currentStyle)
	platform := g.Context.GetPlatform().(*imgui.GLFW)
	glfwWindow = platform.GetWindow()
	applyWindowProperties(glfwWindow)
	mw.SetBgColor(color.RGBA{0, 0, 0, 0})
	mw.SetDropCallback(onDrop)
	mw.Main(loop)
}
