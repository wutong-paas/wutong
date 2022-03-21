#! /bin/bash
# !!! You should run this script on linux/arm64 arch.

export BUILD_ARCH=arm64
export VERSION=v1.0.0-stable-arm64
./release.sh all push
