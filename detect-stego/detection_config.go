package main

// DetectionThresholds contains configurable settings for steganography detection
type DetectionThresholds struct {
	// JPEG Analysis settings
	QuantizationTableModificationThreshold float64 // Threshold for flagging modified quantization tables
	QuantizationTableSimilarityThreshold   float64 // Similarity threshold for comparing to standard tables

	// LSB Analysis settings
	LSBEntropyThreshold          float64 // Entropy value above which LSB distribution is suspicious
	LSBPatternDetectionThreshold float64 // Threshold for detecting LSB patterns

	// Hidden text settings
	TextDetectionConfidenceThreshold float64 // Confidence threshold for reporting hidden text
	MinimumSuspiciousTextLength      int     // Minimum length for suspicious text

	// Statistical analysis settings
	StatisticalAnomalyThreshold float64 // Threshold for statistical anomaly detection
}

// DefaultDetectionConfig returns the default detection configuration
func DefaultDetectionConfig() DetectionThresholds {
	return DetectionThresholds{
		QuantizationTableModificationThreshold: 0.7,  // More conservative than before
		QuantizationTableSimilarityThreshold:   0.85, // 85% similarity to standard tables is acceptable

		LSBEntropyThreshold:          0.999, // Only flag almost perfect entropy
		LSBPatternDetectionThreshold: 0.8,   // Higher threshold for pattern detection

		TextDetectionConfidenceThreshold: 0.8, // More strict threshold for text detection
		MinimumSuspiciousTextLength:      10,  // Text must be at least 10 chars

		StatisticalAnomalyThreshold: 0.7, // Higher threshold for statistical anomalies
	}
}

// CurrentConfig holds the active detection configuration
var CurrentConfig = DefaultDetectionConfig()

// Initialize applies custom configuration settings
func (c *DetectionThresholds) Initialize(filename string) error {
	// Here we could load configuration from a file if needed
	// For now we'll just use the defaults
	return nil
}
