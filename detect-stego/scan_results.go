package main

// ScanResults tracks findings from the steganography scan
type ScanResults struct {
	Filename string

	// JPEG analysis results
	ModifiedQuantizationTables bool    // Deprecated - kept for compatibility
	QuantizationConfidence     float64 // Deprecated - kept for compatibility
	JpegQualityEstimate        int

	// Hidden text results
	HiddenTextFound bool
	DetectedTexts   []string
	TextConfidence  float64

	// LSB analysis results
	LSBAnomaliesFound bool
	LSBEntropy        float64
	LSBPatterns       []string
	LSBConfidence     float64

	// Statistical analysis
	StatisticalAnomalyScore float64
	AnomalyDetails          string

	// Image characteristics
	ImageComplexity float64
	Results         []ScanResult
	TotalFiles      int
	CleanFiles      int
	Suspicious      int
	ConfirmedC2     int

	// False positive analysis
	FalsePositiveLikelihood float64

	// Error tracking
	Errors []string

	Level       DetectionLevel
	Findings    []Finding
	Description string
}
