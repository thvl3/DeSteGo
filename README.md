# SteGoC2
## HackUSU Project - Cyber Category
Created by Ethan Hulse, Connor Dedic, Mckay Thompson, Nathaniel Clark

SteGoC2 is a detection tool built to find C2 traffic embedded into social media images with steganography. The project is built with Go.

We should be able to input a url to a social media post or account and analyze it for any embedded data that shouldn't be there.
The program should tell us what data was found and verify if it is or isnt C2 traffic. The program will be able to display plaintext C2 commands.

The initial version of this project will not source images directly from social media. To download them use a tool like gallery-dl and then point the program to the image directory. Future versions will either use both programs wrapped in a bash script or run both in a docker container. 

# Scan a directory of images
`./detect-stego -dir /path/to/images`

# Download and scan from a single URL
`./detect-stego -url https://example.com/image.png`

# Download and scan from a file containing URLs
`./detect-stego -urlfile urls.txt`

# Specify custom output directory for downloads
`./detect-stego -urlfile urls.txt -outdir custom_folder`