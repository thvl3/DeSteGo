package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
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

// ExtractData tries to read a 32-bit big-endian length, then that many bytes, from the image.
// It reads `mask.RBits` from R, `mask.GBits` from G, etc., in the given bitOrder (LSBFirst or MSBFirst).
func ExtractData(img image.Image, mask ChannelMask, bitOrder BitOrder) ([]byte, error) {
	totalBits := mask.RBits + mask.GBits + mask.BBits + mask.ABits
	if totalBits <= 0 {
		return nil, errors.New("channel mask has 0 total bits")
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

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
			}
		}
	}

	if !foundLength {
		return nil, errors.New("not enough pixels to read 32-bit length")
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length == 0 || length > 100_000_000 {
		return nil, fmt.Errorf("invalid length: %d", length)
	}

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
		}
	}

	if dataBitsRead < dataBitsNeeded {
		return nil, errors.New("not enough bits to read the entire payload")
	}

	return dataBuf, nil
}

// ExtractLSBNoLength extracts LSB data without assuming a length prefix
// It reads all LSBs from the image according to the mask and returns the raw byte data
// This is useful for steganographic methods that don't store a length prefix
func ExtractLSBNoLength(img image.Image, mask ChannelMask, bitOrder BitOrder) []byte {
	bounds := img.Bounds()
	bytesBuilder := bytes.Buffer{}
	currentByte := byte(0)
	bitCount := 0
	zeroCount := 0

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

	return data[:i+1]
}

// BruteForceLSB enumerates 0..3 bits in R/G/B/A plus LSB vs. MSB
// to find possible hidden data. Returns all found results.
// Modified to be more efficient and targeted in its approach.
func BruteForceLSB(img image.Image) []BruteForceResult {
	var results []BruteForceResult

	// Use a more targeted approach with common steganography bit patterns first
	commonMasks := []ChannelMask{
		{RBits: 1, GBits: 0, BBits: 0, ABits: 0}, // R channel only
		{RBits: 0, GBits: 1, BBits: 0, ABits: 0}, // G channel only
		{RBits: 0, GBits: 0, BBits: 1, ABits: 0}, // B channel only
		{RBits: 1, GBits: 1, BBits: 1, ABits: 0}, // All RGB channels
	}

	// First try common patterns with length-prefix extraction
	for _, mask := range commonMasks {
		for _, order := range []BitOrder{LSBFirst, MSBFirst} {
			data, err := ExtractData(img, mask, order)
			if err == nil && len(data) > 0 && (IsASCIIPrintable(data) || ComputeEntropy(data) > 6.5) {
				results = append(results, BruteForceResult{
					Mask:  mask,
					Order: order,
					Data:  data,
				})
			}
		}
	}

	// If we didn't find anything, or want to be exhaustive,
	// try the no-length extraction method for the common patterns
	for _, mask := range commonMasks {
		for _, order := range []BitOrder{LSBFirst, MSBFirst} {
			data := ExtractLSBNoLength(img, mask, order)
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
				}
			}
		}
	}

	// Only try more complex patterns if nothing was found with the common ones
	if len(results) == 0 {
		// For demonstration, 1..2 bits in each channel. Full 0..3 range can be very slow.
		for rBits := 0; rBits <= 2; rBits++ {
			for gBits := 0; gBits <= 2; gBits++ {
				for bBits := 0; gBits <= 2; bBits++ {
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

					// Try extraction with both length prefix and without
					for _, order := range []BitOrder{LSBFirst, MSBFirst} {
						// Try with length prefix
						data, err := ExtractData(img, mask, order)
						if err == nil && len(data) > 0 && (IsASCIIPrintable(data) || ComputeEntropy(data) > 6.5) {
							results = append(results, BruteForceResult{
								Mask:  mask,
								Order: order,
								Data:  data,
							})
							continue // Go to the next order
						}

						// Try without length prefix
						data = ExtractLSBNoLength(img, mask, order)
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
				}
			}
		}
	}

	return results
}
