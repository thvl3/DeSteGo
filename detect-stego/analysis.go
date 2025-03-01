package main

import (
	"fmt"
	"image"
	"strings"
)

func analyzeChiSquare(img image.Image, result *ScanResult) {
	chiR := ChiSquareLSB(img, 'R')
	chiG := ChiSquareLSB(img, 'G')
	chiB := ChiSquareLSB(img, 'B')

	avgChi := (chiR + chiG + chiB) / 3.0

	if avgChi < 0.5 {
		result.AddFinding(
			"Highly uniform LSB distribution",
			9,
			Suspicious,
			fmt.Sprintf("Chi-square avg=%.4f (R=%.4f, G=%.4f, B=%.4f)", avgChi, chiR, chiG, chiB),
		)
	} else if avgChi > 10.0 {
		result.AddFinding(
			"Abnormal LSB distribution",
			7,
			Suspicious,
			fmt.Sprintf("Chi-square avg=%.4f (R=%.4f, G=%.4f, B=%.4f)", avgChi, chiR, chiG, chiB),
		)
	}
}

func analyzeLSB(img image.Image, result *ScanResult) {
	// ... implement LSB analysis with new result system ...
}

func analyzeJSLSB(img image.Image, result *ScanResult) {
	if DetectJSLSB(img) {
		message, err := TryExtractJSLSB(img)
		if err == nil && len(message) > 0 {
			if strings.Contains(strings.ToLower(message), "beacon") ||
				strings.Contains(strings.ToLower(message), "callback") {
				result.AddFinding(
					"JavaScript C2 beacon detected",
					10,
					ConfirmedC2,
					fmt.Sprintf("Extracted JS: %s", message),
				)
			} else {
				result.AddFinding(
					"JavaScript steganography detected",
					8,
					Suspicious,
					fmt.Sprintf("Extracted JS: %s", message),
				)
			}
		}
	}

	// Additional entropy analysis
	dist := AnalyzeLSBDistribution(img)
	if dist.Entropy > 0.95 || dist.Entropy < 0.6 {
		result.AddFinding(
			"Abnormal LSB entropy distribution",
			6,
			Suspicious,
			fmt.Sprintf("Entropy=%.4f", dist.Entropy),
		)
	}
}

func analyzeJPEG(img image.Image, filename string, result *ScanResult) {
	// ... implement JPEG analysis with new result system ...
}
