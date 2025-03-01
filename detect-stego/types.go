package main

// DetectionLevel represents the confidence level of a finding
type DetectionLevel int

const (
	Clean DetectionLevel = iota
	Suspicious
	ConfirmedC2
)

// Finding represents a single detected issue in a file
type Finding struct {
	Description string         // Short description of what was found
	Confidence  int            // Confidence level from 1-10
	Level       DetectionLevel // Clean, Suspicious, or ConfirmedC2
	Details     string         // Additional details about the finding
}

// ScanResult contains all findings for a single file
type ScanResult struct {
	Filename                   string
	ModifiedQuantizationTables bool
	QuantizationConfidence     float64
	JpegQualityEstimate        int
	HiddenTextFound            bool
	DetectedTexts              []string
	TextConfidence             float64
	LSBAnomaliesFound          bool
	LSBEntropy                 float64
	LSBPatterns                []string
	LSBConfidence              float64
	StatisticalAnomalyScore    float64
	AnomalyDetails             string
	ImageComplexity            float64
	FalsePositiveLikelihood    float64
	Findings                   []Finding
	Level                      DetectionLevel
	Description                string
	Errors                     []string
}

// AddFinding adds a new finding to the scan result
func (sr *ScanResult) AddFinding(description string, confidence int, level DetectionLevel, details string) {
	sr.Findings = append(sr.Findings, Finding{
		Description: description,
		Confidence:  confidence,
		Level:       level,
		Details:     details,
	})

	// Update the overall level based on the highest severity finding
	if level > sr.Level {
		sr.Level = level
	}
}
