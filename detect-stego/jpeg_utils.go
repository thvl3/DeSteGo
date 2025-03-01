package main

// StegResults contains steganography analysis results
type StegResults struct {
	JStegProbability    float64
	F5Probability       float64
	OutGuessProbability float64
	Details             map[string]string
}
