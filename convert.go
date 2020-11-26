package moshpit

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strconv"

	"golang.org/x/net/context"
)

// ConvertToAvi uses ffmpeg to convert the input file
// into a mute AVI file for datamoshing.
// The encoding quality of the output file is determined
// by the quality parameter, with 0.0 being the lowest
// and 1.0 being highest possible quality setting.
// If iFrameIndices is not nil, automatic I-Frame generation
// is disabled and I-Frames are placed at the given frame indices.
// The encoding progress is frequently written to the
// progress channel as a value between 0.0 and 1.0.
// The error channel is closed when processing is finished.
func ConvertToAvi(ctx context.Context, ffmpegPath string,
	ffmpegLogPath string, inputFile string, outputFile string, quality float64,
	iFrameIndices []uint64, progressChan chan<- float64,
	errorChan chan<- error) {

	if filepath.Ext(outputFile) != ".avi" {
		errorChan <- errors.New("output file must have the .avi extension")
		close(errorChan)
		return
	}

	if quality < 0 || quality > 1 {
		errorChan <- errors.New("quality setting must be a value between 0 and 1")
		close(errorChan)
		return
	}

	// the ffmpeg quality setting ranges from 0 to 31,
	// with 0 being the best quality.
	ffmpegQuality := uint64(math.Round(31.0 * (1 - quality)))

	// construct ffmpeg arguments
	args := []string{
		"-i", inputFile,
		// disable audio to avoid audio frames
		// messing with the datamoshing to perform
		"-an",
		// set output quality to desired value
		"-q", strconv.FormatUint(ffmpegQuality, 10),
		// force overwrite the output file
		// to avoid the command line prompt before execution
		"-y",
	}

	if iFrameIndices != nil {
		// disable automatic I-Frame generation by setting the
		// requested I-Frame interval to the maximum possible value
		args = append(args, "-g", strconv.FormatInt(math.MaxInt32, 10))

		// experimental mode is required for keyframe intervals larger than 600
		// https://github.com/FFmpeg/FFmpeg/blob/c1b282dc74d801f943db282e89f7729e90897669/libavcodec/mpegvideo_enc.c#L371
		args = append(args, "-strict", "experimental")

		// construct force_key_frames expression
		// that sets I-Frames at the specified frame indices.
		// For each desired keyframe index, it checks
		// whether the current frame index n is equal to it,
		// and adds the results together, which is equivalent
		// to a logical or.
		expr := "expr:"
		for i, frame := range iFrameIndices {
			if i != 0 {
				expr += "+"
			}
			expr += fmt.Sprintf("eq(n,%d)", frame)
		}
		args = append(args, "-force_key_frames", expr)
	}

	// append output file as last argument
	args = append(args, outputFile)

	runFFmpeg(ctx, ffmpegPath, args, ffmpegLogPath, progressChan, nil, errorChan)
}

// ConvertToMp4 uses ffmpeg to convert the input file
// into an MP4 file, taking the audio stream from another file.
// If the sound file path is empty, no audio is added to the output file.
// The encoding quality of the output file is determined
// by the quality parameter, with 0.0 being the lowest
// and 1.0 being highest possible quality setting.
// The encoding progress is frequently written to the
// progress channel as a value between 0.0 and 1.0.
// Any errors encountered are sent to the error channel.
// The error channel is closed when processing is finished.
func ConvertToMp4(ctx context.Context, ffmpegPath string,
	ffmpegLogPath string, aviFile string, soundFile string,
	outputFile string, quality float64,
	progressChan chan<- float64, errorChan chan<- error) {

	if filepath.Ext(outputFile) != ".mp4" {
		errorChan <- errors.New("output file must have the .mp4 extension")
		close(errorChan)
		return
	}

	if quality < 0 || quality > 1 {
		errorChan <- errors.New("quality setting must be a value between 0 and 1")
		close(errorChan)
		return
	}

	// the ffmpeg quality setting ranges from 0 to 31,
	// with 0 being the best quality.
	ffmpegQuality := uint64(math.Round(31.0 * (1 - quality)))

	// construct ffmpeg arguments
	args := []string{
		"-i", aviFile,
	}

	if soundFile != "" {
		args = append(args, "-i", soundFile)

		// take the video stream from the input file,
		// and the audio stream from the sound file

		// makeworld: Question mark was added to support videos with
		// no audio stream. See this issue: https://github.com/CrushedPixel/moshpit/issues/1

		args = append(args, "-map", "0:v:0", "-map", "1:a:0?")

		// the mp4 format requires the aac format for audio streams.
		// use a high bitrate to ensure high-quality audio
		args = append(args, "-c:a", "aac", "-b:a", "320k")
	}

	// set output quality to desired value
	args = append(args, "-q", strconv.FormatUint(ffmpegQuality, 10))

	// speed up encoding at the cost of a larger file size
	args = append(args, "-preset", "ultrafast")

	// force overwrite the output file
	// to avoid the command line prompt before execution
	args = append(args, "-y")

	// append output file as last argument
	args = append(args, outputFile)

	runFFmpeg(ctx, ffmpegPath, args, ffmpegLogPath, progressChan, nil, errorChan)
}
