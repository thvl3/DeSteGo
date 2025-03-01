package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
)

// ChannelMask defines how many bits to read from each channel: R, G, B, A.
type ChannelMask struct {
	RBits int // e.g., 0..3
	GBits int
	BBits int
	ABits int
}

// ExtractData attempts to parse a 4-byte (32-bit) length plus payload from the image
// using the specified ChannelMask for reading bits in the order R->G->B->A.
func ExtractData(img image.Image, mask ChannelMask) ([]byte, error) {
	totalBits := mask.RBits + mask.GBits + mask.BBits + mask.ABits
	if totalBits <= 0 {
		return nil, errors.New("channel mask has 0 total bits")
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Step 1: Extract 32 bits for the length
	lengthBuf := make([]byte, 4)
	lengthBitsRead := 0
	byteIndex := 0
	bitInByte := 0

	// Helper function to append bits into lengthBuf or dataBuf
	writeBit := func(dst []byte, bit uint8) {
		dst[byteIndex] = (dst[byteIndex] << 1) | bit
		bitInByte++
		if bitInByte == 8 {
			bitInByte = 0
			byteIndex++
		}
	}

	neededLengthBits := 32
	foundLength := false

	// Read bits from each pixel in R->G->B->A channels.
outerLengthLoop:
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if lengthBitsRead >= neededLengthBits {
				foundLength = true
				break outerLengthLoop
			}
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			R := uint8(r >> 8)
			G := uint8(g >> 8)
			B := uint8(b >> 8)
			A := uint8(a >> 8)

			// R bits
			for i := 0; i < mask.RBits; i++ {
				if lengthBitsRead >= neededLengthBits {
					foundLength = true
					break
				}
				bit := (R >> i) & 1
				writeBit(lengthBuf, bit)
				lengthBitsRead++
			}

			// G bits
			for i := 0; i < mask.GBits && lengthBitsRead < neededLengthBits; i++ {
				bit := (G >> i) & 1
				writeBit(lengthBuf, bit)
				lengthBitsRead++
			}

			// B bits
			for i := 0; i < mask.BBits && lengthBitsRead < neededLengthBits; i++ {
				bit := (B >> i) & 1
				writeBit(lengthBuf, bit)
				lengthBitsRead++
			}

			// A bits
			for i := 0; i < mask.ABits && lengthBitsRead < neededLengthBits; i++ {
				bit := (A >> i) & 1
				writeBit(lengthBuf, bit)
				lengthBitsRead++
			}
		}
	}

	if !foundLength {
		return nil, errors.New("not enough pixels to read 32-bit length")
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length == 0 || length > 50_000_000 {
		return nil, fmt.Errorf("invalid length %d (0 or too large)", length)
	}

	// Step 2: read 'length' bytes of data = length*8 bits
	dataBuf := make([]byte, length)
	dataBitsNeeded := int(length) * 8
	dataBitsRead := 0
	byteIndex = 0
	bitInByte = 0

	// We'll do a second pass to skip the first 32 bits in the same pattern.
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
					bit := (ch.val >> i) & 1
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

// BruteForceLSB tries multiple channel/bit configurations (e.g., 0..2 bits in R/G/B)
// to see if a hidden message can be extracted. Returns all found messages.
func BruteForceLSB(img image.Image) []struct {
	Mask ChannelMask
	Data []byte
} {
	var results []struct {
		Mask ChannelMask
		Data []byte
	}

	// Example brute force: up to 2 bits in R/G/B.
	// Expand or modify as needed (add alpha, up to 3 bits, etc.).
	for rBits := 0; rBits <= 2; rBits++ {
		for gBits := 0; gBits <= 2; gBits++ {
			for bBits := 0; bBits <= 2; bBits++ {
				if rBits+gBits+bBits == 0 {
					continue
				}
				mask := ChannelMask{RBits: rBits, GBits: gBits, BBits: bBits, ABits: 0}

				data, err := ExtractData(img, mask)
				if err == nil && len(data) > 0 {
					results = append(results, struct {
						Mask ChannelMask
						Data []byte
					}{
						Mask: mask,
						Data: data,
					})
				}
			}
		}
	}

	return results
}
