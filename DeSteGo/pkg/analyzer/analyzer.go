package analyzer

import (
	"image"

	"DeSteGo/pkg/models"
)

// AnalysisOptions holds configuration options for analysis
type AnalysisOptions struct {
	Verbose bool
	Format  string
	Extract bool
	// Additional options can be added as needed
}

// FileAnalyzer is the interface that all file analyzers must implement
type FileAnalyzer interface {
	// CanAnalyze checks if this analyzer can handle the given format
	CanAnalyze(format string) bool

	// Analyze performs analysis on a file and returns results
	Analyze(filePath string, options AnalysisOptions) (*models.AnalysisResult, error)

	// Name returns the name of the analyzer
	Name() string

	// Description returns a detailed description of what the analyzer does
	Description() string

	// SupportedFormats returns a list of file formats this analyzer supports
	SupportedFormats() []string
}

// ImageAnalyzer is an interface for analyzers that work with image files
type ImageAnalyzer interface {
	FileAnalyzer

	// AnalyzeImage performs analysis directly on an image object
	AnalyzeImage(img image.Image, options AnalysisOptions) (*models.AnalysisResult, error)
}

// BaseAnalyzer provides common functionality for analyzers
type BaseAnalyzer struct {
	name        string
	description string
	formats     []string
}

// NewBaseAnalyzer creates a new BaseAnalyzer
func NewBaseAnalyzer(name, description string, formats []string) BaseAnalyzer {
	return BaseAnalyzer{
		name:        name,
		description: description,
		formats:     formats,
	}
}

// Name returns the analyzer name
func (b *BaseAnalyzer) Name() string {
	return b.name
}

// Description returns the analyzer description
func (b *BaseAnalyzer) Description() string {
	return b.description
}

// SupportedFormats returns the supported formats
func (b *BaseAnalyzer) SupportedFormats() []string {
	return b.formats
}

// CanAnalyze checks if the analyzer supports the given format
func (b *BaseAnalyzer) CanAnalyze(format string) bool {
	for _, f := range b.formats {
		if f == format {
			return true
		}
	}
	return false
}
