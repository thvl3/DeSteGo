package grab

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Usage: GrabFromURL(url)
//Make sure the url is a full url path, as a string

// generate a random filename
func GenerateFilename(dir string) (string, error) {
	fp_buffer := make([]byte, 16)
	_, errr := rand.Read(fp_buffer)
	if errr != nil {
		fmt.Println("Error: Could not generate random filename (basic_grab.go)")
	}
	filename := hex.EncodeToString(fp_buffer)
	fullpath := dir + filename + ".jpg"
	// check its stats with os; if it doesn't return an error (meaning the file exists), run it back to ensure no duplicates
	if _, err := os.Stat(fullpath); err == nil {
		return GenerateFilename(dir)
	}
	return fullpath, nil
}

// main grab function, meant to use with direct image urls
func GrabFromURL(url string) {
	// make an http request to given url
	response, err := http.Get(url)

	if err != nil {
		fmt.Println("Error: HTTP request unsuccessful (basic_grab.go)")
	}
	defer response.Body.Close()

	// use a read function to read the bytes from the body slice into the pic_data variable
	pic_data, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error: Could not read image data from HTTP body (basic_grab.go)")
	}

	// generate random name and store the file in the test directory
	output_file, _ := GenerateFilename("../test/")
	os.WriteFile(output_file, pic_data, 0666)
}
