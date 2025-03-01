package main

// DetectionLevel indicates how suspicious a file is
type DetectionLevel int

const (
	Clean DetectionLevel = iota
	Suspicious
	ConfirmedC2
)

// Finding represents a single suspicious element found in a file
type Finding struct {
	Description string
	Confidence  int // 1-10
	Level       DetectionLevel
	Details     string
}

// ScanResult contains all findings for a single file
type ScanResult struct {
	Filename string
	Findings []Finding
	Level    DetectionLevel // Highest level among all findings
}

// ScanResults holds results for multiple files
type ScanResults struct {
	Results     []ScanResult
	TotalFiles  int
	CleanFiles  int
	Suspicious  int
	ConfirmedC2 int
}

// AddFinding adds a new finding to the scan result
func (r *ScanResult) AddFinding(desc string, confidence int, level DetectionLevel, details string) {
	r.Findings = append(r.Findings, Finding{
		Description: desc,
		Confidence:  confidence,
		Level:       level,
		Details:     details,
	})

	// Update the overall level if this finding has a higher level
	if level > r.Level {
		r.Level = level
	}
}
