package analyzer

import (
	"image"

	"DeSteGo/pkg/models"
)

/*
Analyzer.go contains the interface and base implementation for file analyzers.
FileAnalyzer: interface defines the methods that all file analyzers must implement.
ImageAnalyzer: interface extends the FileAnalyzer interface and adds a method for analyzing images directly.
BaseAnalyzer: struct provides common functionality for analyzers, such as name, description, and supported formats.
AnalysisOptions: struct holds configuration options for analysis, such as verbosity, output format, and extraction.
AnalysisResult: struct contains the results of a steganography analysis, including file type, detection score, confidence, possible algorithm, details, findings, recommendations, extraction hints, analysis time, and duration.
Finding struct: represents a specific detection or discovery during analysis, including a description, confidence, and details.
ExtractionHint: struct provides guidance for data extraction, including an algorithm, confidence, and parameters.
ExtractionResult: struct contains the results of an extraction attempt, including success, file type, algorithm, data type, extracted data, data size, details, and output files.
BaseAnalyzer struct: and the FileAnalyzer interface are used by specific analyzers to provide consistent functionality and structure.
*/

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
