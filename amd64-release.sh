#! /bin/bash
# !!! You should run this script on linux/amd64 arch.

export VERSION=v1.1.0-stable
./release.sh all push

# build latest mesh-data-panel and init-probe
# VERSION=latest ./release.sh mesh-data-panel push
# VERSION=latest ./release.sh init-probe push