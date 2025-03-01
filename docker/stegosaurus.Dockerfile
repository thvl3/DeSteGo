FROM debian:bookworm

# Set non-interactive mode to avoid prompts during installation
ENV DEBIAN_FRONTEND=noninteractive

# Update package list, install dependencies, and clean up
RUN apt-get update && \
    apt-get install -y wget git gccgo python3-pip sudo && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Set Go alternative
RUN update-alternatives --set go /usr/bin/go-5 || true

# Set working directory
WORKDIR /repos

# Download the specific binary file for gallery-dl
RUN wget https://github.com/mikf/gallery-dl/releases/download/v1.28.5/gallery-dl.bin -O /repos/gallery-dl.bin
RUN git clone https://github.com/thvl3/SteGOC2.git
# Install gallery-dl, forcing installation even in an externally managed environment
RUN pip3 install --no-cache-dir --break-system-packages gallery-dl

# Set executable permissions
RUN chmod +x /repos/gallery-dl.bin

# Set the default command
CMD ["/bin/bash"]

