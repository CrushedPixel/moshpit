package main

import (
	"bytes"
	"fmt"
	"github.com/k0kubun/go-ansi"
	"github.com/mitchellh/colorstring"
	"github.com/schollz/progressbar"
	"regexp"
	"strings"
)

type floatProgressBar struct {
	*progressbar.ProgressBar

	resolution int
	current    int

	buf           *bytes.Buffer
	maxLineLength int
}

func newDefaultFloatProgressBar(description string) *floatProgressBar {
	return newFloatProgressBar(10000,
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription(colorstring.Color(description)),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        colorstring.Color("[green]="),
			SaucerHead:    colorstring.Color("[green]>"),
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}

func newFloatProgressBar(resolution int, options ...progressbar.Option) *floatProgressBar {
	if resolution < 1 {
		panic("resolution must be a positive value")
	}

	// make the ProgressBar write its output into the bytes.Buffer,
	// so we can inspect and output it to stdout manually
	buf := new(bytes.Buffer)
	options = append(options, progressbar.OptionSetWriter(buf))

	return &floatProgressBar{
		ProgressBar: progressbar.NewOptions(resolution, options...),
		resolution:  resolution,
		buf:         buf,
	}
}

func (p *floatProgressBar) SetProgress(progress float64) {
	if progress < 0 || progress > 1 {
		panic("progress value out of bounds")
	}

	p.buf.Reset()

	toAdd := int(progress*float64(p.resolution)) - p.current
	if toAdd == 0 {
		p.RenderBlank()
	} else {
		p.Add(toAdd)
		p.current += toAdd
	}

	p.writeRendered()
}

func (p *floatProgressBar) RenderBlank() {
	p.buf.Reset()
	p.ProgressBar.RenderBlank()
	p.writeRendered()
}

// regex matching ansi escape codes
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func (p *floatProgressBar) writeRendered() {
	bs := p.buf.String()

	// measure the length of the written progress bar
	// without the ansi escape sequences, to be able
	// to overwrite it with spaces when erasing.
	l := len(ansiRegex.ReplaceAllString(bs, ""))
	if l > p.maxLineLength {
		p.maxLineLength = l
	}
	ansi.Print(bs)
}

func (p *floatProgressBar) Erase() {
	if p.maxLineLength == 0 {
		p.maxLineLength++
	}
	// overwrites the current line with spaces
	// and moves the cursor back to line start.
	// this is required for compatibility with cmd.exe,
	// which does not seem to support ansi.EraseInLine(2)
	fmt.Printf("\r%s\r", strings.Repeat(" ", p.maxLineLength-1))
}
