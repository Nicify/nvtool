package main

import (
	g "github.com/AllenDang/giu"

	"github.com/AllenDang/giu/imgui"
	"github.com/sqweek/dialog"
)

func loadFont() {
	fonts := g.Context.IO().Fonts()
	fontPath := "./iosevka-regular.ttf"
	fonts.AddFontFromFileTTFV(fontPath, 16, imgui.DefaultFontConfig, fonts.GlyphRangesChineseSimplifiedCommon())
}

func selectInputPath() string {
	path, _ := dialog.File().Filter("video file", "mp4", "mkv", "mov", "flv", "avi").Load()
	return path
}

func selectOutputPath() string {
	path, _ := dialog.File().Filter("video file", "mp4", "mkv", "mov", "flv", "avi").Save()
	return path
}
