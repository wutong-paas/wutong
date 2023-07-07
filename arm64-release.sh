#! /bin/bash
# !!! You should run this script on linux/arm64 arch.

export BUILD_ARCH=arm64
export VERSION=v1.3.0-arm64
./release.sh all push

# build latest mesh-data-panel and init-probe
# VERSION=latest-arm64 ./release.sh mesh-data-panel push
# VERSION=latest-arm64 ./release.sh init-probe push