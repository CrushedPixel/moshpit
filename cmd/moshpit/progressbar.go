package main

import (
	"bytes"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v2"
)

type floatProgressBar struct {
	*progressbar.ProgressBar

	resolution int
	current    int

	buf *bytes.Buffer
}

func newDefaultFloatProgressBar(description string) *floatProgressBar {
	return newFloatProgressBar(10000,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
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

func (p *floatProgressBar) Clear() {
	p.buf.Reset()
	p.ProgressBar.Clear()
	p.writeRendered()
}

func (p *floatProgressBar) writeRendered() {
	ansi.Print(p.buf.String())
}
