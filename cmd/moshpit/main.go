package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/c-bata/go-prompt"
	"github.com/crushedpixel/moshpit"
	"github.com/k0kubun/go-ansi"
	"github.com/mitchellh/colorstring"
	"github.com/satori/go.uuid"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var ffmpegPathFlag = flag.String("ffmpeg", "ffmpeg", "path to ffmpeg executable")
var ffmpegLogFlag = flag.String("log", "", "path to ffmpeg log output")

const (
	commandScenes = "scenes"
	commandMosh   = "mosh"
	commandExit   = "exit"
)

func main() {
	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s [options] <input_file>\n", os.Args[0])
		os.Exit(1)
	}

	inputFilePath, err := filepath.Abs(os.Args[len(os.Args)-1])
	if err != nil {
		fmt.Printf("Error parsing input file path: %s\n", err.Error())
		os.Exit(1)
	}

	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Printf("Error opening input file: %s\n", err.Error())
		os.Exit(1)
	}

	// create a context that is cancelled when SIGINT is received
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-c:
			cancel()
		}
	}()

	var ffmpegLogPath string
	if *ffmpegLogFlag != "" {
		ffmpegLogPath, err = filepath.Abs(*ffmpegLogFlag)
		if err != nil {
			fmt.Printf("Error parsing log file path: %s\n", err.Error())
			os.Exit(1)
		}
	}

	promptLoop(ctx, inputFile, *ffmpegPathFlag, ffmpegLogPath)
}

func promptLoop(ctx context.Context, file *os.File, ffmpegPath string, ffmpegLogPath string) {
	completer := promptCompleter(nil)
	var sceneTimes []moshpit.VideoTime

	inputChan := make(chan string)

	// make sure to show cursor before exiting
	defer ansi.CursorShow()

	for {
		go func() {
			ansi.CursorShow()
			if runtime.GOOS == "windows" {
				// on windows, use a plain input prompt,
				// as the ANSI codes from the prompt package
				// are not well supported
				fmt.Print("> ")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				inputChan <- scanner.Text()
			} else {
				inputChan <- prompt.Input("> ", completer)
			}
			ansi.CursorHide()
		}()

		select {
		case input := <-inputChan:
			spl := strings.Split(input, " ")
			command := spl[0]
			args := spl[1:]

			switch strings.ToLower(command) {
			case commandScenes:
				var err error
				sceneTimes, err = cmdScenes(ctx, ffmpegPath, ffmpegLogPath, file, args)
				select {
				case <-ctx.Done():
					return
				default:
				}
				if err != nil {
					fmt.Printf("Error: %s\n", err.Error())
				}

				// update the prompt completer
				// to suggest the newly found scene times
				completer = promptCompleter(sceneTimes)
			case commandMosh:
				err := cmdMosh(ctx, ffmpegPath, ffmpegLogPath, file, sceneTimes, args)
				select {
				case <-ctx.Done():
					return
				default:
				}
				if err != nil {
					fmt.Printf("Error: %s\n", err.Error())
				}
			case commandExit:
				return
			case "":
				// do nothing
			default:
				fmt.Printf("Unknown command: \"%s\"\n", command)
			}

		case <-ctx.Done():
			return
		}
	}
}

func promptCompleter(sceneTimes []moshpit.VideoTime) prompt.Completer {
	commands := []prompt.Suggest{
		{Text: commandScenes, Description: "Finds scene changes in the video file"},
		{Text: commandMosh, Description: "Applies a datamoshing effect to the video file at the given timestamps, and writes them to an output file"},
		{Text: commandExit, Description: "Exits moshpit"},
	}

	return prompt.Completer(func(doc prompt.Document) []prompt.Suggest {
		before := doc.TextBeforeCursor()
		wordsBefore := strings.Split(before, " ")
		// the command being entered is the text until the first space
		commandBefore := wordsBefore[0]
		if len(wordsBefore) == 1 {
			return prompt.FilterHasPrefix(commands, commandBefore, true)
		}

		switch strings.ToLower(commandBefore) {
		case commandMosh:
			if len(wordsBefore) == 2 {
				// TODO: autocomplete output file
			} else {
				// after the output file, suggest all scene change frame indices

				var suggestions []prompt.Suggest
				if len(sceneTimes) > 0 {
					suggestions = append(suggestions, prompt.Suggest{Text: "all", Description: "Mosh all found scene changes"})
					for _, sceneTime := range sceneTimes {
						suggestions = append(suggestions, prompt.Suggest{
							Text:        strconv.FormatUint(sceneTime.Frame, 10),
							Description: sceneTime.Timecode(),
						})
					}
				}
				return prompt.FilterHasPrefix(suggestions, wordsBefore[len(wordsBefore)-1], true)
			}
		}

		return nil
	})
}

func cmdScenes(ctx context.Context, ffmpegPath string, ffmpegLogPath string, file *os.File, args []string) ([]moshpit.VideoTime, error) {
	if len(args) != 1 {
		return nil, errors.New("usage: scenes <threshold>")
	}

	threshold, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return nil, errors.New("threshold must be a valid floating point number")
	}

	sceneTimeChan := make(chan moshpit.VideoTime)
	progressChan := make(chan float64)
	errorChan := make(chan error)
	go moshpit.FindScenes(ctx, ffmpegPath, ffmpegLogPath, file.Name(), threshold, sceneTimeChan, progressChan, errorChan)

	// always write a newline before returning to ensure
	// following command line output is written in the next line
	defer fmt.Println("")

	bar := newDefaultFloatProgressBar("Detecting scene changes...")
	bar.RenderBlank()
	var sceneTimes []moshpit.VideoTime
	for {
		select {
		case err, ok := <-errorChan:
			if !ok {
				bar.Clear()
				ansi.Printf(colorstring.Color("Found [green]%d[reset] scene changes."), len(sceneTimes))
				if len(sceneTimes) == 0 {
					fmt.Println()
					ansi.Print(colorstring.Color("Try using a lower threshold value."))
				}
				return sceneTimes, nil
			}
			return sceneTimes, err
		case sceneTime := <-sceneTimeChan:
			sceneTimes = append(sceneTimes, sceneTime)

			// erase progress bar
			bar.Clear()

			colorstring.Fprintf(ansi.NewAnsiStdout(),
				"Found scene change at [cyan]%s[reset] (frame [red]%d[reset])",
				sceneTime.Timecode(), sceneTime.Frame)
			fmt.Println()

			// rewrite progress bar
			bar.RenderBlank()
		case progress := <-progressChan:
			bar.SetProgress(progress)
		}
	}
}

func cmdMosh(ctx context.Context, ffmpegPath string, ffmpegLogPath string, file *os.File,
	sceneTimes []moshpit.VideoTime, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: mosh <output> <frame> [...]")
	}

	// parse and validate output file path
	outputFilePath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("error parsing output file path: %s", err.Error())
	}

	if filepath.Ext(outputFilePath) != ".mp4" {
		return errors.New("output file must have the .mp4 extension")
	}

	// parse and validate frame indices to mosh
	var moshFrames []uint64
	for _, arg := range args[1:] {
		if arg == "all" {
			// add all previously detected scene changes
			// to the slice of frames to mosh
			if len(sceneTimes) == 0 {
				fmt.Printf(`WARNING: option "all": no scene changes were previously found\n`)
				continue
			}
			for _, sceneTime := range sceneTimes {
				moshFrames = append(moshFrames, sceneTime.Frame)
			}
		} else {
			frame, err := strconv.ParseUint(arg, 10, 64)
			if err != nil {
				fmt.Printf(`WARNING: option "%s" is not a valid frame index\n`, arg)
				continue
			}
			moshFrames = append(moshFrames, frame)
		}
	}

	if len(moshFrames) == 0 {
		return errors.New("no valid frames to mosh were specified")
	}

	// keep track of execution time
	startTime := time.Now()

	aviFileName, err := convertToAvi(ctx, ffmpegPath, ffmpegLogPath, file, moshFrames)
	if err != nil {
		return err
	}
	defer os.Remove(aviFileName)

	moshedFileName, err := moshAvi(ctx, aviFileName, moshFrames)
	if err != nil {
		return err
	}
	defer os.Remove(moshedFileName)

	if err := bake(ctx, ffmpegPath, ffmpegLogPath, file.Name(), moshedFileName, outputFilePath); err != nil {
		return err
	}

	fmt.Printf(colorstring.Color("Moshing took [green]%s[reset].\n"), time.Since(startTime).Round(time.Second))
	return nil
}

func convertToAvi(ctx context.Context, ffmpegPath string, ffmpegLogPath string, file *os.File, moshFrames []uint64) (string, error) {
	// convert input file to AVI with I-Frames at the given frame indices.
	// generate file name for temporary AVI file
	uid, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("could not generate temp file name: %s", err.Error())
	}
	aviFileName := path.Join(os.TempDir(), fmt.Sprintf("%s.avi", uid.String()))

	progressChan := make(chan float64)
	errorChan := make(chan error)
	go moshpit.ConvertToAvi(ctx, ffmpegPath, ffmpegLogPath, file.Name(), aviFileName, 1, moshFrames, progressChan, errorChan)

	// always write a newline before returning to ensure
	// following command line output is written in the next line
	defer fmt.Println("")

	bar := newDefaultFloatProgressBar("[cyan][1/3][reset] Writing moshable file...")
	bar.RenderBlank()
	for {
		select {
		case err, ok := <-errorChan:
			if !ok {
				// processing has finished
				bar.Clear()
				ansi.Print(colorstring.Color("[cyan][1/3][reset] Wrote AVI file for moshing."))
				return aviFileName, nil
			}
			os.Remove(aviFileName)
			return "", fmt.Errorf("error writing AVI file: %s", err.Error())
		case progress := <-progressChan:
			bar.SetProgress(progress)
		}
	}
}

func moshAvi(ctx context.Context, aviFileName string, moshFrames []uint64) (string, error) {
	// remove I-Frames from the AVI file for the datamoshing effect
	aviFile, err := os.Open(aviFileName)
	if err != nil {
		return "", fmt.Errorf("could not open AVI file for datamoshing: %s", err.Error())
	}
	defer aviFile.Close()

	// create output file for moshed AVI
	uid, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("could not generate temp file name: %s", err.Error())
	}
	moshedFileName := path.Join(os.TempDir(), fmt.Sprintf("%s.avi", uid.String()))
	moshedFile, err := os.Create(moshedFileName)
	if err != nil {
		return "", fmt.Errorf("could not create AVI file for datamoshing: %s", err.Error())
	}
	defer moshedFile.Close()

	processedChan := make(chan uint64)
	errorChan := make(chan error)
	go moshpit.RemoveFrames(ctx, aviFile, moshedFile, moshFrames, processedChan, errorChan)

	// always write a newline before returning to ensure
	// following command line output is written in the next line
	defer fmt.Println("")

	bar := newDefaultFloatProgressBar("[cyan][2/3][reset] Moshing AVI file...")
	bar.RenderBlank()
	for {
		select {
		case err, ok := <-errorChan:
			if !ok {
				// processing has finished
				bar.Clear()
				ansi.Print(colorstring.Color("[cyan][2/3][reset] Moshed AVI file."))
				return moshedFileName, nil
			}
			os.Remove(moshedFileName)
			return "", fmt.Errorf("error moshing AVI file: %s", err.Error())
		case _ = <-processedChan:
			bar.SetProgress(0.5) // TODO: proper progress
		}
	}
}

func bake(ctx context.Context, ffmpegPath string, ffmpegLogPath string, originalFileName string,
	moshedFileName string, outputFileName string) error {

	// convert avi to output mp4
	progressChan := make(chan float64)
	errorChan := make(chan error)
	go moshpit.ConvertToMp4(ctx, ffmpegPath, ffmpegLogPath,
		moshedFileName, originalFileName, outputFileName,
		1, progressChan, errorChan)

	// always write a newline before returning to ensure
	// following command line output is written in the next line
	defer fmt.Println("")

	bar := newDefaultFloatProgressBar("[cyan][3/3][reset] Baking output file...")
	bar.RenderBlank()
	for {
		select {
		case err, ok := <-errorChan:
			if !ok {
				bar.Clear()
				ansi.Print(colorstring.Color("[cyan][3/3][reset] Baked output file."))
				return nil
			}
			return fmt.Errorf("error writing output file: %s", err.Error())
		case progress := <-progressChan:
			bar.SetProgress(progress)
		}
	}
}
