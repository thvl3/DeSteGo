package main

import (
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

// BruteForceLSB enumerates 0..3 bits in R/G/B/A plus LSB vs. MSB
// to find possible hidden data. Returns all found results.
func BruteForceLSB(img image.Image) []BruteForceResult {
	var results []BruteForceResult

	// For demonstration, 0..3 bits in each channel. That can be very slow on large images.
	for rBits := 0; rBits <= 3; rBits++ {
		for gBits := 0; gBits <= 3; gBits++ {
			for bBits := 0; bBits <= 3; bBits++ {
				for aBits := 0; aBits <= 3; aBits++ {
					if rBits+gBits+bBits+aBits == 0 {
						continue
					}
					mask := ChannelMask{
						RBits: rBits,
						GBits: gBits,
						BBits: bBits,
						ABits: aBits,
					}

					for _, order := range []BitOrder{LSBFirst, MSBFirst} {
						data, err := ExtractData(img, mask, order)
						if err == nil && len(data) > 0 {
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

	return results
}
