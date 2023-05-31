#!/bin/bash

# Set the program name
PROGRAM=wtctl

# Set the program version
VERSION=1.1.0

# Set the output directory
OUTPUT_DIR=./bin

# Set the target platforms
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

# Create the output directory if it doesn't exist
mkdir -p ${OUTPUT_DIR}

# Loop through each target platform and compile the program
for PLATFORM in "${PLATFORMS[@]}"; do
    # Split the platform into OS and architecture
    OSARCH=(${PLATFORM//\// })
    OS=${OSARCH[0]}
    ARCH=${OSARCH[1]}

    # Set the output file name
    OUTPUT_FILE=${OUTPUT_DIR}/${PROGRAM}-${VERSION}-${OS}-${ARCH}

    if [ "${OS}" = "windows" ]; then
        OUTPUT_FILE=${OUTPUT_FILE}.exe
    fi

    # Compile the program for the target platform
    env GOOS=${OS} GOARCH=${ARCH} go build -o ${OUTPUT_FILE} .

    # Add the executable bit to the output file
    chmod +x ${OUTPUT_FILE}
done
