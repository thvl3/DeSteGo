package models

import (
	"time"
)

// AnalysisResult contains the results of a steganography analysis
type AnalysisResult struct {
	FileType          string                 `json:"fileType"`
	Filename          string                 `json:"filename"`
	DetectionScore    float64                `json:"detectionScore"` // 0.0-1.0 where 1.0 means definitely contains steganography
	Confidence        float64                `json:"confidence"`     // 0.0-1.0 confidence in the detection score
	PossibleAlgorithm string                 `json:"possibleAlgorithm"`
	Details           map[string]interface{} `json:"details"`
	Findings          []Finding              `json:"findings"`
	Recommendations   []string               `json:"recommendations"`
	ExtractionHints   []ExtractionHint       `json:"extractionHints"`
	AnalysisTime      time.Time              `json:"analysisTime"`
	AnalysisDuration  time.Duration          `json:"analysisDuration"`
}

// Finding represents a specific detection or discovery during analysis
type Finding struct {
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"` // 0.0-1.0
	Details     string  `json:"details"`
}

// ExtractionHint provides guidance for data extraction
type ExtractionHint struct {
	Algorithm  string                 `json:"algorithm"`
	Confidence float64                `json:"confidence"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ExtractionResult contains the results of an extraction attempt
type ExtractionResult struct {
	Success       bool                   `json:"success"`
	FileType      string                 `json:"fileType"`
	Algorithm     string                 `json:"algorithm"`
	DataType      string                 `json:"dataType"`      // text, binary, image, etc.
	ExtractedData []byte                 `json:"extractedData"` // The raw extracted data
	DataSize      int                    `json:"dataSize"`
	Details       map[string]interface{} `json:"details"`
	OutputFiles   []string               `json:"outputFiles"` // Paths to any saved output files
	MimeType      string                 `json:"mimeType"`
}

// AddFinding adds a finding to the analysis result
func (r *AnalysisResult) AddFinding(description string, confidence float64, details string) {
	r.Findings = append(r.Findings, Finding{
		Description: description,
		Confidence:  confidence,
		Details:     details,
	})
}

// AddExtractionHint adds an extraction hint to the analysis result
func (r *AnalysisResult) AddExtractionHint(algorithm string, confidence float64, parameters map[string]interface{}) {
	r.ExtractionHints = append(r.ExtractionHints, ExtractionHint{
		Algorithm:  algorithm,
		Confidence: confidence,
		Parameters: parameters,
	})
}

// GetHighestConfidenceAlgorithm returns the extraction algorithm with highest confidence
func (r *AnalysisResult) GetHighestConfidenceAlgorithm() (string, float64, map[string]interface{}) {
	if len(r.ExtractionHints) == 0 {
		return "", 0.0, nil
	}

	best := r.ExtractionHints[0]
	for _, hint := range r.ExtractionHints {
		if hint.Confidence > best.Confidence {
			best = hint
		}
	}

	return best.Algorithm, best.Confidence, best.Parameters
}
