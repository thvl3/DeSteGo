package main

// DQT represents a JPEG quantization table
type DQT struct {
	Tq byte   // Table identifier (0-3)
	Q  []byte // Quantization values (64 bytes for 8x8 block)
}

// NOTE: We're keeping these functions for information gathering only,
// but we won't use them for steganography detection since they produce
// too many false positives

// analyzeQuantizationTables examines quantization tables but no longer flags them as suspicious
// Returns false by default to indicate tables are not considered modified for stego detection
func analyzeQuantizationTables(dqt []DQT) (bool, float64, string) {
	// Always return false - we no longer use quantization tables as an indicator
	return false, 0.0, "Quantization table analysis disabled - too many false positives"
}

// getStandardQuantizationTables returns standard JPEG quantization tables for information
func getStandardQuantizationTables() map[string]DQT {
	tables := make(map[string]DQT)

	// JPEG standard luminance table
	tables["JPEG Standard Luminance"] = DQT{
		Tq: 0,
		Q: []byte{
			16, 11, 10, 16, 24, 40, 51, 61,
			12, 12, 14, 19, 26, 58, 60, 55,
			14, 13, 16, 24, 40, 57, 69, 56,
			14, 17, 22, 29, 51, 87, 80, 62,
			18, 22, 37, 56, 68, 109, 103, 77,
			24, 35, 55, 64, 81, 104, 113, 92,
			49, 64, 78, 87, 103, 121, 120, 101,
			72, 92, 95, 98, 112, 100, 103, 99,
		},
	}

	// JPEG standard chrominance table
	tables["JPEG Standard Chrominance"] = DQT{
		Tq: 1,
		Q: []byte{
			17, 18, 24, 47, 99, 99, 99, 99,
			18, 21, 26, 66, 99, 99, 99, 99,
			24, 26, 56, 99, 99, 99, 99, 99,
			47, 66, 99, 99, 99, 99, 99, 99,
			99, 99, 99, 99, 99, 99, 99, 99,
			99, 99, 99, 99, 99, 99, 99, 99,
			99, 99, 99, 99, 99, 99, 99, 99,
			99, 99, 99, 99, 99, 99, 99, 99,
		},
	}

	return tables
}
