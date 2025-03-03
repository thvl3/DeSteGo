package analyzer

import (
	"sync"
)

// Registry is a container for all available analyzers
type Registry struct {
	analyzers map[string][]FileAnalyzer
	mu        sync.RWMutex
}

// NewRegistry creates a new analyzer registry
func NewRegistry() *Registry {
	return &Registry{
		analyzers: make(map[string][]FileAnalyzer),
	}
}

// Register adds an analyzer to the registry
func (r *Registry) Register(analyzer FileAnalyzer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, format := range analyzer.SupportedFormats() {
		r.analyzers[format] = append(r.analyzers[format], analyzer)
	}
}

// GetAnalyzersForFormat returns all analyzers that support the given format
func (r *Registry) GetAnalyzersForFormat(format string) []FileAnalyzer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.analyzers[format]
}

// GetSupportedFormats returns a list of all supported formats
func (r *Registry) GetSupportedFormats() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var formats []string
	for format := range r.analyzers {
		formats = append(formats, format)
	}

	return formats
}
