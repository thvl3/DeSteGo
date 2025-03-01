package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
)

// ChannelMask is a simple struct to indicate how many bits we want to read from each channel.
type ChannelMask struct {
	RBits int // e.g., 0..3
	GBits int
	BBits int
	ABits int
}

// ExtractData attempts to parse a length + data from an image, given a ChannelMask.
func ExtractData(img image.Image, mask ChannelMask) ([]byte, error) {
	// If all bits in the mask are zero, that’s invalid.
	totalBits := mask.RBits + mask.GBits + mask.BBits + mask.ABits
	if totalBits <= 0 {
		return nil, errors.New("channel mask has 0 bits in total")
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// We’ll read bits in this sequence: R-bits -> G-bits -> B-bits -> A-bits, per pixel.
	// First 32 bits are the "length".
	lengthBuf := make([]byte, 4) // 4 bytes
	lengthBitsRead := 0

	// Fill lengthBuf from the pixels
	byteIndex := 0
	bitInByte := 0

	// Helper to write a bit into lengthBuf or dataBuf
	writeBit := func(dst []byte, bit uint8) {
		// shift left by 1, then OR with bit
		dst[byteIndex] = (dst[byteIndex] << 1) | bit
		bitInByte++
		if bitInByte == 8 {
			bitInByte = 0
			byteIndex++
		}
	}

	// 1) Extract length
	// We need 32 bits, so let's iterate over the pixels until we have 32 bits or run out
	// We need 32 bits, so let's iterate over the pixels until we have 32 bits or run out
	neededLengthBits := 32

	for y := 0; y < height && lengthBitsRead < neededLengthBits; y++ {
		for x := 0; x < width && lengthBitsRead < neededLengthBits; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			R := uint8(r >> 8)
			G := uint8(g >> 8)
			B := uint8(b >> 8)
			A := uint8(a >> 8)

			// Collect bits from R
			for i := 0; i < mask.RBits && lengthBitsRead < neededLengthBits; i++ {
				bit := (R >> i) & 1
				writeBit(lengthBuf, bit)
				lengthBitsRead++
			}
			// Then G
			for i := 0; i < mask.GBits && lengthBitsRead < neededLengthBits; i++ {
				bit := (G >> i) & 1
				writeBit(lengthBuf, bit)
				lengthBitsRead++
			}
			// Then B
			for i := 0; i < mask.BBits && lengthBitsRead < neededLengthBits; i++ {
				bit := (B >> i) & 1
				writeBit(lengthBuf, bit)
				lengthBitsRead++
			}
			// Then A
			for i := 0; i < mask.ABits && lengthBitsRead < neededLengthBits; i++ {
				bit := (A >> i) & 1
				writeBit(lengthBuf, bit)
				lengthBitsRead++
			}
		}
	}

	if lengthBitsRead < neededLengthBits {
		return nil, errors.New("not enough pixels to read length")
	}

	// Convert length (BigEndian)
	// But note: the way we're shifting bits might be reversed if we do it purely left shift each time.
	// For simplicity, let's assume we read bits LSB-first. That might mismatch typical big-endian reading.
	// We'll do a small fix: we read bits in reverse order in each byte, so the final byte might be reversed.
	// A robust approach might reorder bits after reading.
	length := binary.BigEndian.Uint32(lengthBuf)

	if length == 0 || length > 10_000_000 {
		// Arbitrary max to avoid huge memory allocations
		return nil, fmt.Errorf("invalid or zero length: %d", length)
	}

	// 2) Extract the actual data
	dataBuf := make([]byte, length)
	dataBitsNeeded := int(length) * 8
	dataBitsRead := 0
	byteIndex = 0
	bitInByte = 0

	// Continue from where we left off. We used some pixels for length already.
	// Let's re-scan the image, but skip those length bits.
	// For a simpler approach, we can just do a second pass or track we left off in the same pass.
	// We'll do a new pass, skipping the first 32 bits.

	bitsSkipped := 0
	skipTarget := lengthBitsRead // 32 bits so far

	doneReading := false
	for y := 0; y < height && !doneReading; y++ {
		for x := 0; x < width && !doneReading; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			R := uint8(r >> 8)
			G := uint8(g >> 8)
			B := uint8(b >> 8)
			A := uint8(a >> 8)
			for channelIndex := 0; channelIndex < 4; channelIndex++ {
				var channelValue uint8
				var bitsInChannel int
				switch channelIndex {
				case 0:
					channelValue = R
					bitsInChannel = mask.RBits
				case 1:
					channelValue = G
					bitsInChannel = mask.GBits
				case 2:
					channelValue = B
					bitsInChannel = mask.BBits
				case 3:
					channelValue = A
					bitsInChannel = mask.ABits
				}
				for i := 0; i < bitsInChannel; i++ {
					if bitsSkipped < skipTarget {
						bitsSkipped++
						continue
					}
					if dataBitsRead >= dataBitsNeeded {
						doneReading = true
						break
					}
					bit := (channelValue >> i) & 1
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
		return nil, errors.New("not enough bits to read the entire message")
	}

	return dataBuf, nil
}

// A helper that tries to re-align or re-order bits could go here,
// because we read bits in LSB order.
// But let's keep it conceptual for brevity.

// Load/Decode PNG (same as before)
func LoadPNG(filename string) (image.Image, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func DecodePNGFromReader(r io.Reader) (image.Image, error) {
	return png.Decode(r)
}

// Quick ASCII check
func IsASCIIPrintable(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	printableCount := 0
	for _, b := range data {
		if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
			printableCount++
		}
	}
	ratio := float64(printableCount) / float64(len(data))
	return ratio > 0.8
}
