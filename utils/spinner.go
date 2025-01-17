package utils

import (
	spinner2 "github.com/briandowns/spinner"
	"sync"
	"time"
)

// This package controls whether there's a spinner displayed or not. To be used in longer running processes.
// Mainly, it's a wrapper around briandowns/spinner.

var showSpinner bool
var spinnerMx sync.Mutex

var spinner *spinner2.Spinner

func SetShowSpinner(spin bool) {
	spinnerMx.Lock()
	defer spinnerMx.Unlock()
	showSpinner = spin
}

func ShowSpinner() {
	spinnerMx.Lock()
	defer spinnerMx.Unlock()
	if showSpinner {
		if spinner != nil {
			if spinner.Active() {
				return
			}
		} else {
			spinner = spinner2.New(spinner2.CharSets[9], 100*time.Millisecond)
		}
		spinner.Start()
	}
}

func HideSpinner() {
	spinnerMx.Lock()
	defer spinnerMx.Unlock()
	if showSpinner && spinner != nil {
		if spinner.Active() {
			spinner.Stop()
		}
		spinner = nil
	}
}

func init() {
	spinnerMx = sync.Mutex{}
}
