package extractor

import (
	"image"

	"DeSteGo/pkg/models"
)

// ExtractionOptions contains configuration for extraction process
type ExtractionOptions struct {
	OutputDir      string
	AlgorithmHints []string
	Parameters     map[string]interface{}
	Password       string
	Verbose        bool
}

// DataExtractor is the interface that all extractors must implement
type DataExtractor interface {
	// CanExtract checks if this extractor can handle the given format
	CanExtract(format string) bool

	// Extract attempts to extract hidden data from a file
	Extract(filePath string, options ExtractionOptions) (*models.ExtractionResult, error)

	// Name returns the name of the extractor
	Name() string

	// SupportedFormats returns formats this extractor supports
	SupportedFormats() []string

	// SupportedAlgorithms returns steganography algorithms this extractor handles
	SupportedAlgorithms() []string
}

// ImageExtractor is an interface for extractors that work with image files
type ImageExtractor interface {
	DataExtractor

	// ExtractFromImage extracts data directly from an image object
	ExtractFromImage(img image.Image, options ExtractionOptions) (*models.ExtractionResult, error)
}

// BaseExtractor provides common functionality for extractors
type BaseExtractor struct {
	name       string
	formats    []string
	algorithms []string
}

// NewBaseExtractor creates a new BaseExtractor
func NewBaseExtractor(name string, formats []string, algorithms []string) BaseExtractor {
	return BaseExtractor{
		name:       name,
		formats:    formats,
		algorithms: algorithms,
	}
}

// Name returns the extractor name
func (b *BaseExtractor) Name() string {
	return b.name
}

// SupportedFormats returns the supported formats
func (b *BaseExtractor) SupportedFormats() []string {
	return b.formats
}

// SupportedAlgorithms returns the supported algorithms
func (b *BaseExtractor) SupportedAlgorithms() []string {
	return b.algorithms
}

// CanExtract checks if the extractor supports the given format
func (b *BaseExtractor) CanExtract(format string) bool {
	for _, f := range b.formats {
		if f == format {
			return true
		}
	}
	return false
}
