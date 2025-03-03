package extractor

import (
	"sync"
)

// Registry is a container for all available extractors
type Registry struct {
	extractors map[string][]DataExtractor
	mu         sync.RWMutex
}

// NewRegistry creates a new extractor registry
func NewRegistry() *Registry {
	return &Registry{
		extractors: make(map[string][]DataExtractor),
	}
}

// Register adds an extractor to the registry
func (r *Registry) Register(extractor DataExtractor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, format := range extractor.SupportedFormats() {
		r.extractors[format] = append(r.extractors[format], extractor)
	}
}

// GetExtractorsForFormat returns all extractors that support the given format
func (r *Registry) GetExtractorsForFormat(format string) []DataExtractor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.extractors[format]
}

// GetExtractorByName finds an extractor with the given name
func (r *Registry) GetExtractorByName(name string, format string) DataExtractor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	extractors := r.extractors[format]
	for _, e := range extractors {
		if e.Name() == name {
			return e
		}
	}

	return nil
}

// GetSupportedFormats returns all formats that have registered extractors
func (r *Registry) GetSupportedFormats() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var formats []string
	for format := range r.extractors {
		formats = append(formats, format)
	}

	return formats
}
