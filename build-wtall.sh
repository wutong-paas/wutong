#!/bin/bash
set -o errexit

build_items=(api chaos gateway monitor mq webcli worker eventlog init-probe mesh-data-panel grctl node)
# build_items=(eventlog)

buildallbinaries() {
	for item in "${build_items[@]}"; do
		local BUILDER_DIR="./cmd/$item"
		local CGOENABLED=1
		local CC="gcc"
		local build_args="-w -s -X github.com/wutong-paas/wutong/cmd.version=${release_desc}"

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

		if [ "$item" = "gateway" ]; then
			CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -ldflags "${build_args}" -o /output/bin/wutong-"$item" $BUILDER_DIR
		else
			if [ "${CC}" = "gcc" ]; then
				CGO_ENABLED=$CGOENABLED GOOS=$TARGETOS GOARCH=$TARGETARCH CC=$CC go build -ldflags "${build_args}" -o /output/bin/wutong-"$item" $BUILDER_DIR
			else
				CGO_ENABLED=$CGOENABLED GOOS=$TARGETOS GOARCH=$TARGETARCH CC=$CC go build -buildmode=c-shared -o /output/bin/wutong-"$item" $BUILDER_DIR
			fi
		fi

	done
}

buildallbinaries