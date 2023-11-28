#! /bin/bash
# !!! You should run this script on linux/arm64 arch.

export BUILD_ARCH=arm64
export VERSION=$(git describe --tags --always --dirty)-arm64
./single-arch-build.sh all push