package moshpit

import (
	"bytes"
	"golang.org/x/net/context"
	"io"
)

// RemoveFrames writes a copy of the AVI data from the input reader
// to the output writer, replacing the frames at the given indices
// with the following frame.
// Any errors encountered are sent to the error channel.
// The error channel is closed when processing is finished.
func RemoveFrames(ctx context.Context, input io.Reader, output io.Writer,
	framesToRemove []uint64, processedChan chan<- uint64, errorChan chan<- error) {

	defer close(errorChan)
	r := AviScanner(input)

	// counter of how many frames to duplicate
	duplicate := 0
	i := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if r.Scan() {
				frame := r.Bytes()
				if i == 0 {
					// write the original frames
					// without incrementing the counter
					// until the first I-Frame is encountered -
					// all frames before that are header frames
					if _, err := output.Write(frame); err != nil {
						errorChan <- err
						return
					}

					if bytes.Compare(frame[5:8], iframePrefix) == 0 {
						// we found the first I-Frame - increment
						// the counter to stop the search and begin moshing
						i++
					}

					continue
				}

				if contains(framesToRemove, uint64(i)) {
					duplicate++
				} else {
					duplicate++
					for duplicate > 0 {
						if _, err := output.Write(frame); err != nil {
							errorChan <- err
							return
						}
						duplicate--
					}
				}

				if i >= 0 {
					processedChan <- uint64(i)
				}

				i++
			} else {
				if err := r.Err(); err != nil {
					errorChan <- err
				}
				return
			}
		}
	}
}

func contains(values []uint64, x uint64) bool {
	for _, val := range values {
		if val == x {
			return true
		}
	}
	return false
}
