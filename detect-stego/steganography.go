package main

import (
    "encoding/binary"
    "errors"
    "image"
    "image/png"
    "io"
    "os"
)

// ExtractMessageLSB attempts to read a length (4 bytes) + data from the LSB of the red channel.
func ExtractMessageLSB(img image.Image) ([]byte, error) {
    bounds := img.Bounds()

    // Step 1: Read 4 bytes (32 bits) for the hidden data length
    var lengthBuf [4]byte
    dataIndex := 0
    bitIndex := 0

    for y := bounds.Min.Y; y < bounds.Max.Y && dataIndex < 4; y++ {
        for x := bounds.Min.X; x < bounds.Max.X && dataIndex < 4; x++ {
            r, _, _, _ := img.At(x, y).RGBA()
            R := uint8(r >> 8) // Convert [0..65535] to [0..255]
            bit := R & 1       // Grab the least significant bit

            // Shift the current byte left by 1 and add the bit
            lengthBuf[dataIndex] = (lengthBuf[dataIndex] << 1) | bit

            bitIndex++
            if bitIndex == 8 {
                bitIndex = 0
                dataIndex++
            }
        }
    }

    length := binary.BigEndian.Uint32(lengthBuf[:])
    if length == 0 {
        return nil, errors.New("no embedded data found or length is zero")
    }

    // Step 2: Read 'length' bytes of hidden data
    ciphertext := make([]byte, length)
    dataIndex = 0
    bitIndex = 0
    bitsNeeded := int(length) * 8
    bitsRead := 0
    // We used 32 bits (4 bytes) for the length, so skip those 32 pixels
    skipPixels := 32
    pixelCount := 0

    for y := bounds.Min.Y; y < bounds.Max.Y && bitsRead < bitsNeeded; y++ {
        for x := bounds.Min.X; x < bounds.Max.X && bitsRead < bitsNeeded; x++ {
            if pixelCount < skipPixels {
                pixelCount++
                continue
            }

            r, _, _, _ := img.At(x, y).RGBA()
            R := uint8(r >> 8)
            bit := R & 1

            ciphertext[dataIndex] = (ciphertext[dataIndex] << 1) | bit
            bitIndex++
            if bitIndex == 8 {
                bitIndex = 0
                dataIndex++
            }

            bitsRead++
            pixelCount++
        }
    }

    return ciphertext, nil
}

// LoadPNG loads a PNG from disk
func LoadPNG(filename string) (image.Image, error) {
    f, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    img, err := png.Decode(f)
    if err != nil {
        return nil, err
    }
    return img, nil
}

// DecodePNGFromReader decodes a PNG from an io.Reader.
func DecodePNGFromReader(r io.Reader) (image.Image, error) {
    return png.Decode(r)
}

// IsASCIIPrintable checks if 'data' is mostly ASCII-printable.
func IsASCIIPrintable(data []byte) bool {
    printableCount := 0
    for _, b := range data {
        // "printable" range: [32..126], plus newline, carriage return, tab
        if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
            printableCount++
        }
    }
    // if at least 80% of characters are "printable", treat as ASCII
    ratio := float64(printableCount) / float64(len(data))
    return ratio > 0.8
}

