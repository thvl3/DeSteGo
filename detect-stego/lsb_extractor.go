package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"time"
)

// BitOrder enumerates whether we read bits in LSB order (lowest bits first) or MSB.
type BitOrder int

const (
	LSBFirst BitOrder = iota
	MSBFirst
)

// ChannelMask defines how many bits to read from R, G, B, A.
type ChannelMask struct {
	RBits int
	GBits int
	BBits int
	ABits int
}

// BruteForceResult holds a found payload plus metadata about how it was extracted.
type BruteForceResult struct {
	Mask  ChannelMask
	Order BitOrder
	Data  []byte
}

// ProgressCallback is a function type that receives progress updates
type ProgressCallback func(percentComplete float64, message string)

// NoProgress is a dummy progress callback that does nothing
func NoProgress(_ float64, _ string) {}

// ExtractData tries to read a 32-bit big-endian length, then that many bytes, from the image.
// It reads `mask.RBits` from R, `mask.GBits` from G, etc., in the given bitOrder (LSBFirst or MSBFirst).
// The progress callback is called periodically with updates.
func ExtractData(img image.Image, mask ChannelMask, bitOrder BitOrder, progressCb ProgressCallback) ([]byte, error) {
	if progressCb == nil {
		progressCb = NoProgress
	}

	totalBits := mask.RBits + mask.GBits + mask.BBits + mask.ABits
	if totalBits <= 0 {
		return nil, errors.New("channel mask has 0 total bits")
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	totalPixels := width * height
	pixelsProcessed := 0
	lastProgress := 0.0
	lastProgressTime := time.Now()

	// Step 1: Extract 32 bits for length
	progressCb(0, fmt.Sprintf("Extracting length header (mask: R:%d G:%d B:%d A:%d)",
		mask.RBits, mask.GBits, mask.BBits, mask.ABits))

	// Add progress updates
	updateProgress := func(x, y int, stage string) {
		pixelsProcessed++

		// Only update progress every 1% or at least 500ms to avoid flooding output
		progress := float64(pixelsProcessed) / float64(totalPixels) * 100.0
		if progress-lastProgress >= 1.0 || time.Since(lastProgressTime) > 500*time.Millisecond {
			progressCb(progress, fmt.Sprintf("%s (%.1f%% - mask: R:%d G:%d B:%d A:%d)",
				stage, progress, mask.RBits, mask.GBits, mask.BBits, mask.ABits))
			lastProgress = progress
			lastProgressTime = time.Now()
		}
	}

	// Step 1: Extract 32 bits for length
	lengthBuf := make([]byte, 4)
	lengthBitsRead := 0
	byteIndex := 0
	bitInByte := 0

	// Helper function to shift in the new bit from the right.
	// We'll store the final result in big-endian order, but consistently.
	writeBit := func(dst []byte, bit uint8) {
		// shift left 1, then OR with the new bit
		dst[byteIndex] = (dst[byteIndex] << 1) | bit
		bitInByte++
		if bitInByte == 8 {
			bitInByte = 0
			byteIndex++
		}
	}

	neededLengthBits := 32
	foundLength := false

outerLoop:
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if lengthBitsRead >= neededLengthBits {
				foundLength = true
				break outerLoop
			}
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			R := uint8(r >> 8)
			G := uint8(g >> 8)
			B := uint8(b >> 8)
			A := uint8(a >> 8)

			channels := []struct {
				val  uint8
				bits int
			}{
				{R, mask.RBits},
				{G, mask.GBits},
				{B, mask.BBits},
				{A, mask.ABits},
			}

			for _, ch := range channels {
				for i := 0; i < ch.bits; i++ {
					if lengthBitsRead >= neededLengthBits {
						foundLength = true
						break
					}

					var bit uint8
					if bitOrder == LSBFirst {
						// LSB side => shift out bit i from the right
						bit = (ch.val >> i) & 1
					} else {
						// MSB side => shift out bit i from the left
						shift := (ch.bits - 1) - i
						bit = (ch.val >> shift) & 1
					}
					writeBit(lengthBuf, bit)
					lengthBitsRead++
				}
				if foundLength {
					break
				}
			}
			updateProgress(x, y, "Reading length header")
		}
	}

	if !foundLength {
		return nil, errors.New("not enough pixels to read 32-bit length")
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length == 0 || length > 100_000_000 {
		return nil, fmt.Errorf("invalid length: %d", length)
	}

	progressCb(float64(pixelsProcessed)/float64(totalPixels)*100.0,
		fmt.Sprintf("Found length: %d bytes, extracting data...", length))

	// Step 2: read 'length' bytes => length*8 bits
	dataBuf := make([]byte, length)
	dataBitsNeeded := int(length) * 8
	dataBitsRead := 0
	byteIndex = 0
	bitInByte = 0

	// We'll skip the first 32 bits from the same pattern.
	bitsSkipped := 0
	skipTarget := lengthBitsRead
	doneReading := false

	for y := 0; y < height && !doneReading; y++ {
		for x := 0; x < width && !doneReading; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			R := uint8(r >> 8)
			G := uint8(g >> 8)
			B := uint8(b >> 8)
			A := uint8(a >> 8)

			channels := []struct {
				val  uint8
				bits int
			}{
				{R, mask.RBits},
				{G, mask.GBits},
				{B, mask.BBits},
				{A, mask.ABits},
			}

			for _, ch := range channels {
				for i := 0; i < ch.bits; i++ {
					if bitsSkipped < skipTarget {
						bitsSkipped++
						continue
					}
					if dataBitsRead >= dataBitsNeeded {
						doneReading = true
						break
					}

					var bit uint8
					if bitOrder == LSBFirst {
						bit = (ch.val >> i) & 1
					} else {
						shift := (ch.bits - 1) - i
						bit = (ch.val >> shift) & 1
					}

					dataBuf[byteIndex] = (dataBuf[byteIndex] << 1) | bit
					bitInByte++
					if bitInByte == 8 {
						bitInByte = 0
						byteIndex++
					}
					dataBitsRead++
				}
				if doneReading {
					break
				}
			}
			updateProgress(x, y, fmt.Sprintf("Extracting data (%d/%d bytes)",
				byteIndex, length))
		}
	}

	if dataBitsRead < dataBitsNeeded {
		return nil, errors.New("not enough bits to read the entire payload")
	}

	progressCb(100.0, fmt.Sprintf("Completed extraction: %d bytes", len(dataBuf)))

	return dataBuf, nil
}

// ExtractLSBNoLength extracts LSB data without assuming a length prefix
// It now includes progress reporting via callback
func ExtractLSBNoLength(img image.Image, mask ChannelMask, bitOrder BitOrder, progressCb ProgressCallback) []byte {
	if progressCb == nil {
		progressCb = NoProgress
	}

	bounds := img.Bounds()
	totalPixels := bounds.Dx() * bounds.Dy()
	pixelsProcessed := 0
	lastProgress := 0.0
	lastProgressTime := time.Now()

	progressCb(0, fmt.Sprintf("Starting direct extraction (mask: R:%d G:%d B:%d A:%d)",
		mask.RBits, mask.GBits, mask.BBits, mask.ABits))

	bytesBuilder := bytes.Buffer{}
	currentByte := byte(0)
	bitCount := 0
	zeroCount := 0

	// Add progress reporting
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			R := uint8(r >> 8)
			G := uint8(g >> 8)
			B := uint8(b >> 8)
			A := uint8(a >> 8)

			channels := []struct {
				val  uint8
				bits int
			}{
				{R, mask.RBits},
				{G, mask.GBits},
				{B, mask.BBits},
				{A, mask.ABits},
			}

			for _, ch := range channels {
				for i := 0; i < ch.bits; i++ {
					var bit uint8
					if bitOrder == LSBFirst {
						bit = (ch.val >> i) & 1
					} else {
						shift := (ch.bits - 1) - i
						bit = (ch.val >> shift) & 1
					}

					currentByte = (currentByte << 1) | bit
					bitCount++

					// When we have 8 bits, add the byte to our result
					if bitCount == 8 {
						bytesBuilder.WriteByte(currentByte)

						// Count consecutive zero bytes to detect end of data
						if currentByte == 0 {
							zeroCount++
							// If we have 20 consecutive zeros, assume we're past the end of the message
							if zeroCount >= 20 {
								data := bytesBuilder.Bytes()
								endPos := len(data) - 20
								if endPos <= 0 {
									return nil
								}
								return data[:endPos]
							}
						} else {
							zeroCount = 0
						}

						// Reset for next byte
						currentByte = 0
						bitCount = 0
					}
				}
			}

			pixelsProcessed++

			// Update progress every 1% or 500ms
			progress := float64(pixelsProcessed) / float64(totalPixels) * 100.0
			if progress-lastProgress >= 1.0 || time.Since(lastProgressTime) > 500*time.Millisecond {
				progressCb(progress, fmt.Sprintf("Direct extraction (%.1f%% - %d bytes so far)",
					progress, bytesBuilder.Len()))
				lastProgress = progress
				lastProgressTime = time.Now()
			}
		}
	}

	// Handle any remaining bits
	if bitCount > 0 {
		currentByte = currentByte << (8 - bitCount) // Align to MSB
		bytesBuilder.WriteByte(currentByte)
	}

	// Clean null bytes from the end
	data := bytesBuilder.Bytes()
	i := len(data) - 1
	for i >= 0 && data[i] == 0 {
		i--
	}

	if i < 0 {
		return nil // All zeros, no data
	}

	progressCb(100.0, fmt.Sprintf("Completed direct extraction: %d bytes", len(data)))

	return data[:i+1]
}

// BruteForceLSB enumerates 0..3 bits in R/G/B/A plus LSB vs. MSB
// to find possible hidden data. Returns all found results.
// Modified to be more efficient and targeted in its approach.
// Now with progress reporting
func BruteForceLSB(img image.Image, progressCb ProgressCallback) []BruteForceResult {
	if progressCb == nil {
		progressCb = NoProgress
	}

	var results []BruteForceResult

	progressCb(0.0, "Starting LSB brute force scan...")

	// Calculate total operations for progress tracking
	commonMasks := []ChannelMask{
		{RBits: 1, GBits: 0, BBits: 0, ABits: 0}, // R channel only
		{RBits: 0, GBits: 1, BBits: 0, ABits: 0}, // G channel only
		{RBits: 0, GBits: 0, BBits: 1, ABits: 0}, // B channel only
		{RBits: 1, GBits: 1, BBits: 1, ABits: 0}, // All RGB channels
	}

	totalOperations := len(commonMasks) * 2 * 2 // common masks × with/without length × bit orders
	operationsCompleted := 0

	// First try common patterns with length-prefix extraction
	progressCb(0.0, "Trying common steganography patterns with length prefix...")

	for _, mask := range commonMasks {
		for _, order := range []BitOrder{LSBFirst, MSBFirst} {
			orderName := "LSB first"
			if order == MSBFirst {
				orderName = "MSB first"
			}

			progressCb(float64(operationsCompleted)/float64(totalOperations)*40.0,
				fmt.Sprintf("Testing mask R:%d G:%d B:%d A:%d with %s (with length)",
					mask.RBits, mask.GBits, mask.BBits, mask.ABits, orderName))

			// Use local progress callback for detailed updates within this extraction
			localCb := func(percent float64, msg string) {
				// Scale to a smaller segment of the overall progress
				overallPercent := 40.0 +
					(float64(operationsCompleted)+percent/100.0)/
						float64(totalOperations)*40.0
				progressCb(overallPercent, msg)
			}

			data, err := ExtractData(img, mask, order, localCb)
			operationsCompleted++

			if err == nil && len(data) > 0 && (IsASCIIPrintable(data) || ComputeEntropy(data) > 6.5) {
				results = append(results, BruteForceResult{
					Mask:  mask,
					Order: order,
					Data:  data,
				})

				progressCb(float64(operationsCompleted)/float64(totalOperations)*40.0,
					fmt.Sprintf("Found potential data with mask R:%d G:%d B:%d A:%d (%s, %d bytes)",
						mask.RBits, mask.GBits, mask.BBits, mask.ABits, orderName, len(data)))
			}
		}
	}

	// If we didn't find anything, or want to be exhaustive,
	// try the no-length extraction method for the common patterns
	progressCb(40.0, "Trying direct bit extraction (no length header)...")

	for i, mask := range commonMasks {
		for j, order := range []BitOrder{LSBFirst, MSBFirst} {
			orderName := "LSB first"
			if order == MSBFirst {
				orderName = "MSB first"
			}

			progressCb(40.0+float64(i*2+j)/float64(len(commonMasks)*2)*40.0,
				fmt.Sprintf("Direct extraction: mask R:%d G:%d B:%d A:%d with %s",
					mask.RBits, mask.GBits, mask.BBits, mask.ABits, orderName))

			// Use local progress callback for detailed updates
			localCb := func(percent float64, msg string) {
				// Scale to a smaller segment of the overall progress
				overallPercent := 40.0 +
					(float64(i*2+j)+percent/100.0)/
						float64(len(commonMasks)*2)*40.0
				progressCb(overallPercent, msg)
			}

			data := ExtractLSBNoLength(img, mask, order, localCb)
			operationsCompleted++

			if len(data) > 10 && (IsASCIIPrintable(data) || ComputeEntropy(data) > 6.5) {
				// Check if this is different from what we've already found
				isDuplicate := false
				for _, result := range results {
					if bytes.Equal(result.Data, data) {
						isDuplicate = true
						break
					}
				}

				if !isDuplicate {
					results = append(results, BruteForceResult{
						Mask:  mask,
						Order: order,
						Data:  data,
					})

					progressCb(40.0+float64(i*2+j+1)/float64(len(commonMasks)*2)*40.0,
						fmt.Sprintf("Found potential data with direct extraction R:%d G:%d B:%d A:%d (%s, %d bytes)",
							mask.RBits, mask.GBits, mask.BBits, mask.ABits, orderName, len(data)))
				}
			}
		}
	}

	// Only try more complex patterns if nothing was found with the common ones
	if len(results) == 0 {
		progressCb(80.0, "No results with common patterns, trying advanced combinations...")

		// For demonstration, 1..2 bits in each channel. Full 0..3 range can be very slow.
		totalAdvanced := 0
		for rBits := 0; rBits <= 2; rBits++ {
			for gBits := 0; gBits <= 2; gBits++ {
				for bBits := 0; bBits <= 2; bBits++ {
					// Skip combinations we've already tested or with no bits
					if (rBits == 1 && gBits == 0 && bBits == 0) ||
						(rBits == 0 && gBits == 1 && bBits == 0) ||
						(rBits == 0 && gBits == 0 && bBits == 1) ||
						(rBits == 1 && gBits == 1 && bBits == 1) ||
						(rBits+gBits+bBits == 0) {
						continue
					}
					totalAdvanced++
				}
			}
		}

		currentAdvanced := 0

		for rBits := 0; rBits <= 2; rBits++ {
			for gBits := 0; gBits <= 2; gBits++ {
				for bBits := 0; bBits <= 2; bBits++ {
					// Skip combinations we've already tested
					if (rBits == 1 && gBits == 0 && bBits == 0) ||
						(rBits == 0 && gBits == 1 && bBits == 0) ||
						(rBits == 0 && gBits == 0 && bBits == 1) ||
						(rBits == 1 && gBits == 1 && bBits == 1) {
						continue
					}

					// Skip combinations with no bits
					if rBits+gBits+bBits == 0 {
						continue
					}

					mask := ChannelMask{
						RBits: rBits,
						GBits: gBits,
						BBits: bBits,
						ABits: 0, // Usually Alpha is not used for steganography
					}

					progressCb(80.0+float64(currentAdvanced)/float64(totalAdvanced)*20.0,
						fmt.Sprintf("Advanced testing: mask R:%d G:%d B:%d", rBits, gBits, bBits))

					// Try both with length and without
					for _, order := range []BitOrder{LSBFirst, MSBFirst} {
						// Try with length prefix
						data, err := ExtractData(img, mask, order, progressCb)
						if err == nil && len(data) > 0 && (IsASCIIPrintable(data) || ComputeEntropy(data) > 6.5) {
							results = append(results, BruteForceResult{
								Mask:  mask,
								Order: order,
								Data:  data,
							})
							continue // Go to the next order
						}

						// Try without length prefix
						data = ExtractLSBNoLength(img, mask, order, progressCb)
						if len(data) > 10 && (IsASCIIPrintable(data) || ComputeEntropy(data) > 6.5) {
							// Check for duplicates
							isDuplicate := false
							for _, result := range results {
								if bytes.Equal(result.Data, data) {
									isDuplicate = true
									break
								}
							}

							if !isDuplicate {
								results = append(results, BruteForceResult{
									Mask:  mask,
									Order: order,
									Data:  data,
								})
							}
						}
					}
					currentAdvanced++
				}
			}
		}
	}

	progressCb(100.0, fmt.Sprintf("LSB brute force completed, found %d potential results", len(results)))

	return results
}
