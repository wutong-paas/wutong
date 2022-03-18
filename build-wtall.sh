#!/bin/bash
set -o errexit

# build_items=(api chaos gateway monitor mq webcli worker eventlog init-probe mesh-data-panel grctl node resource-proxy)
# build_items=(api chaos gateway monitor mq webcli worker eventlog init-probe mesh-data-panel grctl node)
build_items=(api chaos gateway monitor mq webcli worker init-probe mesh-data-panel grctl node)
# build_items=(eventlog)

buildallbinaries() {
	for item in "${build_items[@]}"; do
		local BUILDER_DIR="./cmd/$item"
		local CGOENABLED=1
		local CC="gcc"

		if [ "${TARGETARCH}" = "arm64" ]; then
			CC="aarch64-linux-gnu-gcc"
			PKG_CONFIG_PATH=/usr/lib/aarch64-linux-gnu/pkgconfig
		fi

	    if [ "$item" = "chaos" ]; then
			BUILDER_DIR="./cmd/builder"
		elif [ "$item" = "monitor" ]; then
			CGOENABLED=0
			CC=gcc
		fi

		if [ "${CC}" = "gcc" ]; then
	    	CGO_ENABLED=$CGOENABLED GOOS=$TARGETOS GOARCH=$TARGETARCH CC=$CC go build -a -o /output/bin/wutong-"$item" $BUILDER_DIR
	    else
			CGO_ENABLED=$CGOENABLED GOOS=$TARGETOS GOARCH=$TARGETARCH CC=$CC go build -buildmode=c-shared -a -o /output/bin/wutong-"$item" $BUILDER_DIR
		fi

	done
}

buildallbinaries