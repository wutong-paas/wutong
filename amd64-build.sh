#! /bin/bash
# !!! You should run this script on linux/amd64 arch.

export VERSION=$(git describe --tags --always --dirty)-amd64
./single-arch-build.sh all push