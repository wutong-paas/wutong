#!/bin/bash
set -e

# build linux amd64
GOOS=linux GOARCH=amd64 go build -o bin/upgrader-v1.1.0-linux-amd64

# build linux arm64
GOOS=linux GOARCH=arm64 go build -o bin/upgrader-v1.1.0-linux-arm64