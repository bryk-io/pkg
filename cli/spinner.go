package cli

import (
	"time"

	sp "github.com/briandowns/spinner"
)

// Spinner indicator.
type Spinner struct {
	el *sp.Spinner
}

// NewSpinner creates a new spinner indicator instance.
func NewSpinner(opts ...SpinnerOption) *Spinner {
	s := new(Spinner)
	s.el = sp.New(sp.CharSets[11], 100*time.Millisecond)
	s.el.HideCursor = true
	_ = s.el.Color("fgHiBlack")
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start the spinner indicator.
func (s *Spinner) Start() {
	s.el.Start()
}

// Stop the spinner indicator.
func (s *Spinner) Stop() {
	s.el.Stop()
}

// SpinnerOption provide a functional style mechanism to adjust the settings
// when creating a new spinner instance.
type SpinnerOption = func(s *Spinner)

// WithSpinnerColor adjust the color used for the spinner indicator.
// Supported values are: "green", "blue", "yellow" and "red".
func WithSpinnerColor(color string) SpinnerOption {
	return func(s *Spinner) {
		if c, ok := supportedSpinnerColors[color]; ok {
			_ = s.el.Color(c)
		}
	}
}

var supportedSpinnerColors = map[string]string{
	"blue":   "blue",
	"red":    "red",
	"yellow": "yellow",
	"green":  "green",
}
