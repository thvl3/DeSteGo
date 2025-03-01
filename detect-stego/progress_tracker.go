package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ProgressTracker manages multiple concurrent progress bars
type ProgressTracker struct {
	mu           sync.Mutex
	progressBars map[string]*ProgressBar
	output       io.Writer
	lastUpdate   time.Time
}

// ProgressBar represents a single progress operation
type ProgressBar struct {
	ID           string
	Description  string
	Percent      float64
	LastUpdateTS time.Time
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		progressBars: make(map[string]*ProgressBar),
		output:       os.Stdout,
		lastUpdate:   time.Now(),
	}
}

// Start begins tracking a new operation
func (pt *ProgressTracker) Start(id, description string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.progressBars[id] = &ProgressBar{
		ID:           id,
		Description:  description,
		Percent:      0.0,
		LastUpdateTS: time.Now(),
	}

	pt.render()
}

// Update updates the progress of an operation
func (pt *ProgressTracker) Update(id string, percent float64, description string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if bar, exists := pt.progressBars[id]; exists {
		bar.Percent = percent
		if description != "" {
			bar.Description = description
		}
		bar.LastUpdateTS = time.Now()
	} else {
		// Create a new bar if it doesn't exist
		pt.progressBars[id] = &ProgressBar{
			ID:           id,
			Description:  description,
			Percent:      percent,
			LastUpdateTS: time.Now(),
		}
	}

	// Only render if some time has passed since last update
	// to avoid flooding terminal output
	if time.Since(pt.lastUpdate) > 100*time.Millisecond {
		pt.render()
		pt.lastUpdate = time.Now()
	}
}

// Complete marks an operation as complete and removes it from display
func (pt *ProgressTracker) Complete(id string, message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if bar, exists := pt.progressBars[id]; exists {
		bar.Percent = 100.0
		if message != "" {
			bar.Description = message
		}

		// Display completion message
		fmt.Fprintf(pt.output, "\r%s: %s [Complete]\n", id, bar.Description)

		// Remove from tracking
		delete(pt.progressBars, id)
	}

	pt.render()
}

// render displays all progress bars
func (pt *ProgressTracker) render() {
	// Clear previous lines
	if len(pt.progressBars) > 0 {
		fmt.Fprint(pt.output, "\r")
	}

	// Render each progress bar
	for id, bar := range pt.progressBars {
		// Create a progress bar display [=====>    ]
		width := 30
		completed := int(bar.Percent / 100.0 * float64(width))

		progressBar := "["
		for i := 0; i < width; i++ {
			if i < completed {
				progressBar += "="
			} else if i == completed {
				progressBar += ">"
			} else {
				progressBar += " "
			}
		}
		progressBar += "]"

		// Truncate description if too long
		desc := bar.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		// Print the progress bar with ID, percentage and description
		fmt.Fprintf(pt.output, "\r%s: %s %.1f%% %s\n",
			id, progressBar, bar.Percent, desc)
	}

	// Move cursor back up to the first progress bar
	if len(pt.progressBars) > 0 {
		fmt.Fprint(pt.output, strings.Repeat("\033[F", len(pt.progressBars)))
	}
}

// GetProgressCallback returns a callback function for ProgressCallback type
// that updates this tracker
func (pt *ProgressTracker) GetProgressCallback(id string) ProgressCallback {
	return func(percentComplete float64, message string) {
		pt.Update(id, percentComplete, message)

		if percentComplete >= 100.0 {
			pt.Complete(id, message)
		}
	}
}
