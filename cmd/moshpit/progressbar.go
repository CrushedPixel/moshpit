package main

import (
	"github.com/k0kubun/go-ansi"
	"github.com/mitchellh/colorstring"
	"github.com/schollz/progressbar"
)

type floatProgressBar struct {
	*progressbar.ProgressBar

	resolution int
	current    int
}

func newDefaultFloatProgressBar(description string) *floatProgressBar {
	return newFloatProgressBar(10000,
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription(colorstring.Color(description)),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
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

	return &floatProgressBar{
		ProgressBar: progressbar.NewOptions(resolution, options...),
		resolution:  resolution,
	}
}

func (p *floatProgressBar) SetProgress(progress float64) {
	if progress < 0 || progress > 1 {
		panic("progress value out of bounds")
	}

	toAdd := int(progress*float64(p.resolution)) - p.current
	if toAdd == 0 {
		p.RenderBlank()
	} else {
		if err := p.Add(toAdd); err != nil {
			panic(err)
		}
		p.current += toAdd
	}
}

func (p *floatProgressBar) Erase() {
	ansi.EraseInLine(2)
	ansi.CursorHorizontalAbsolute(0)
}
