package moshpit

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// regex extracting the duration value of an input stream.
// the duration has the format HOURS:MINUTES:SECONDS.MILLISECONDS
var ffmpegStreamDurationRegex = regexp.MustCompile(`Duration: ([0-9]*):([0-9]*):([0-9]*).([0-9]*)`)

// regex extracting the out_time_ms value from an ffmpeg progress line.
var ffmpegOutTimeMSRegex = regexp.MustCompile(`out_time_ms=([0-9]*)`)

// runFFmpeg runs ffmpeg with the given arguments,
// frequently sending progress updates to the progress channel.
// If stderrLineChan is not nil, FFmpeg's stderr output is
// written to stderrLineChan line by line.
// Any errors encountered are sent to the error channel.
// The error channel is closed when processing is finished.
func runFFmpeg(ctx context.Context, ffmpegPath string,
	args []string, ffmpegLogPath string,
	progressChan chan<- float64,
	stderrLineChan chan<- string,
	errorChan chan<- error) {

	defer close(errorChan)

	var logFile *os.File
	if ffmpegLogPath != "" {
		var err error
		logFile, err = os.OpenFile(ffmpegLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			errorChan <- fmt.Errorf("error opening log file: %s", err.Error())
			return
		}
		defer logFile.Close()
	}

	args = append(args[:len(args)-1],
		// inject the -progress option before the output file name
		// so we can parse progress information from stdout
		// and supply it to the progress channel.
		// TODO: test on Windows
		"-progress", os.Stdout.Name(),
		args[len(args)-1])

	if logFile != nil {
		if _, err := logFile.WriteString(fmt.Sprintf("Executing %s %s\n", ffmpegPath, strings.Join(args, " "))); err != nil {
			errorChan <- fmt.Errorf("error writing to log file: %s", err.Error())
			return
		}
	}

	cmd := exec.CommandContext(ctx, ffmpegPath, args...)

	// ffmpeg writes its output to stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		errorChan <- err
		return
	}

	// the progress data is written to stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errorChan <- err
		return
	}

	// start the command execution without blocking
	// to be able to read from stdout and stderr
	if err := cmd.Start(); err != nil {
		errorChan <- err
		return
	}

	stderrChan := make(chan string)
	go readLinesToChannel(stderr, stderrChan)

	stdoutChan := make(chan string)
	go readLinesToChannel(stdout, stdoutChan)

	// initially send 0% progress message
	progressChan <- 0

	var duration time.Duration
read:
	for {
		select {
		case line, ok := <-stderrChan:
			if !ok {
				break read
			}
			if logFile != nil {
				if _, err := logFile.WriteString(fmt.Sprintf("%s\n", line)); err != nil {
					errorChan <- fmt.Errorf("error writing to log file: %s", err.Error())
					return
				}
			}

			if stderrLineChan != nil {
				stderrLineChan <- line
			}
			if duration == 0 {
				// look for video duration line
				if m := ffmpegStreamDurationRegex.FindStringSubmatch(line); m != nil {
					hours, err := strconv.ParseInt(m[1], 10, 64)
					if err != nil {
						errorChan <- fmt.Errorf("error parsing duration value: %s", err.Error())
						return
					}
					duration += time.Duration(hours) * time.Hour

					minutes, err := strconv.ParseInt(m[2], 10, 64)
					if err != nil {
						errorChan <- fmt.Errorf("error parsing duration value: %s", err.Error())
						return
					}
					duration += time.Duration(minutes) * time.Minute

					seconds, err := strconv.ParseInt(m[3], 10, 64)
					if err != nil {
						errorChan <- fmt.Errorf("error parsing duration value: %s", err.Error())
						return
					}
					duration += time.Duration(seconds) * time.Second

					millis, err := strconv.ParseInt(m[4], 10, 64)
					if err != nil {
						errorChan <- fmt.Errorf("error parsing duration value: %s", err.Error())
						return
					}
					duration += time.Duration(millis) * time.Millisecond
				}
			}

		case line, ok := <-stdoutChan:
			if !ok {
				stdoutChan = nil
			}
			if m := ffmpegOutTimeMSRegex.FindStringSubmatch(line); m != nil {
				if duration == 0 {
					// we haven't found the input duration value,
					// which should always occur before the -progress output
					errorChan <- errors.New("could not find duration of input file")
					return
				}

				millis, err := strconv.ParseInt(m[1], 10, 64)
				if err != nil {
					errorChan <- fmt.Errorf("error parsing output time value: %s", err.Error())
					return
				}

				p := time.Duration(millis) * time.Microsecond
				// limit the progress to 1.0, as the duration printed
				// by ffmpeg may be a bit inaccurate.
				// we could use ffprobe to get the precise duration of the
				// period, but it's really not worth the hassle.
				progress := math.Min(1, float64(p)/float64(duration))
				progressChan <- progress
			}
		}
	}

	// wait for the ffmpeg command to finish
	if err := cmd.Wait(); err != nil {
		errorChan <- err
		return
	}
}

func readLinesToChannel(reader io.Reader, lineChan chan<- string) {
	r := bufio.NewScanner(reader)
	for r.Scan() {
		lineChan <- r.Text()
	}
	close(lineChan)
}
