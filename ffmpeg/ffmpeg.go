package ffmpeg

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	ffmpegBinary       = "ffmpeg"
	ffmpegPrefix       = []string{"-y", "-hide_banner"}
	ffmpegCmd          *exec.Cmd
	encodingTime       uint
	durationRegexp     = regexp.MustCompile(`Duration: (\d{2}):(\d{2}):(\d{2})\.(\d{2})`)
	encodingTimeRegexp = regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2})\.(\d{2})`)
	isEncodingRegexp   = regexp.MustCompile(`speed=\d+\.\d+x`)
)

// GetDurationFromTimeParams
func GetDurationFromTimeParams(time []string) uint {
	var (
		hour     uint64
		min      uint64
		sec      uint64
		ms       uint64
		duration uint
	)
	hour, err := strconv.ParseUint(time[1], 10, 32)
	if err == nil {
		min, err = strconv.ParseUint(time[2], 10, 32)
	}
	if err == nil {
		sec, err = strconv.ParseUint(time[3], 10, 32)
	}
	if err == nil {
		ms, err = strconv.ParseUint(time[4], 10, 32)
	}
	if err == nil {
		duration = uint(hour*60*60*1000 + min*60*1000 + sec*1000 + ms*10)
	}
	return duration
}

// GetDuration
func GetDuration(inputPath string) (uint, error) {
	cmd := exec.Command(ffmpegBinary, "-i", inputPath)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 0, err
	}
	err = cmd.Start()
	if err != nil {
		return 0, err
	}
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		matches := durationRegexp.FindStringSubmatch(line)
		if len(matches) == 5 {
			duration := GetDurationFromTimeParams(matches)
			return duration, nil
		}
	}
	return 0, scanner.Err()
}

// RunEncode
func RunEncode(inputPath string, outputPath string, args []string, progress *float32, log *string, isEncoding *bool, onUpdate func()) {
	fullDuration, err := GetDuration(inputPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	args = append([]string{"-i", inputPath}, args...)
	args = append(ffmpegPrefix, args...)
	args = append(args, outputPath)
	fmt.Println(args)
	ffmpegCmd = exec.Command(ffmpegBinary, args...)
	stderr, err := ffmpegCmd.StderrPipe()
	if err != nil {
		fmt.Println(err)
	}
	err = ffmpegCmd.Start()
	if err != nil {
		fmt.Println(err)
	}
	var info []string
	scanner := bufio.NewScanner(stderr)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		line := scanner.Text()
		if *isEncoding {
			info = append(info, line)
		}
		matches := encodingTimeRegexp.FindStringSubmatch(line)
		if len(matches) == 5 {
			encodingTime = GetDurationFromTimeParams(matches)
			*progress = float32(encodingTime) / float32(fullDuration)
			onUpdate()
		}
		if isEncodingRegexp.MatchString(line) {
			*isEncoding = true
			logChunk := strings.Join(info, " ")
			logChunk = strings.ReplaceAll(logChunk, "frame=", "\nframe=")
			logChunk = strings.ReplaceAll(logChunk, "= ", "=")
			*log += logChunk
			info = info[:0]
			onUpdate()
		}
	}
}
