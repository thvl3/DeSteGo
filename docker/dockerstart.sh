#!/bin/bash
cat ../steGOsaurus.txt
sudo docker build -f stegosaurus.Dockerfile -t stegosaurus .
sudo docker run -it stegosaurus:latest /bin/bash
