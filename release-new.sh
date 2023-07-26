#!/bin/bash
set -o errexit

# define package name
WORK_DIR=/go/src/github.com/wutong-paas/wutong
BASE_NAME=wutong

GO_VERSION=1.17

GOPROXY=${GOPROXY:-'https://goproxy.io'}

if [ "$DISABLE_GOPROXY" == "true" ]; then
	GOPROXY=
fi
if [ "$DEBUG" ]; then
	set -x
fi
BRANCH=$(git symbolic-ref HEAD 2>/dev/null | cut -d"/" -f 3)
if [ -z "$VERSION" ]; then
	VERSION=$BRANCH-dev
fi

buildTime=$(date +%F-%H)
git_commit=$(git log -n 1 --pretty --format=%h)

release_desc=${VERSION}-${git_commit}-${buildTime}
build_items=(api chaos gateway monitor mq webcli worker eventlog init-probe mesh-data-panel wtctl node resource-proxy)

build::binary() {
	echo "---> build binary:$1"
	home=$(pwd)
	local go_mod_cache="${home}/.cache"
	local AMD64_OUTPATH="./_output/binary/linux/amd64/${BASE_NAME}-$1"
	local ARM64_OUTPATH="./_output/binary/linux/arm64/${BASE_NAME}-$1"
	local DOCKER_PATH="./hack/contrib/docker/$1"
	local build_image="golang:${GO_VERSION}"
	local build_args="-w -s -X github.com/wutong-paas/wutong/cmd.version=${release_desc}"
	local build_dir="./cmd/$1"
	local DOCKERFILE_BASE="Dockerfile.multiarch"
	if [ -f "${DOCKER_PATH}/ignorebuild" ]; then
		return
	fi
	CGO_ENABLED=0
	if [ "$1" = "eventlog" ]; then
		CGO_ENABLED=1
		build_image="wutongpaas/eventlog-builder:v1"
	elif [ "$1" = "gateway" ]; then
		build_image="golang:${GO_VERSION}-alpine"
	fi
	if [ "$1" = "eventlog" ]; then
		docker run --platform=linux/amd64 --rm -e CGO_ENABLED=${CGO_ENABLED} -e GOPROXY=${GOPROXY} -e GOOS=linux -e GOARCH=amd64 -v "${go_mod_cache}":/go/pkg/mod -v "$(pwd)":${WORK_DIR} -w ${WORK_DIR} ${build_image} go build -ldflags "${build_args}" -o "${AMD64_OUTPATH}" ${build_dir}
		docker run --platform=linux/arm64 --rm -e CGO_ENABLED=${CGO_ENABLED} -e GOPROXY=${GOPROXY} -e GOOS=linux -e GOARCH=arm64 -v "${go_mod_cache}":/go/pkg/mod -v "$(pwd)":${WORK_DIR} -w ${WORK_DIR} ${build_image} go build -ldflags "${build_args}" -o "${ARM64_OUTPATH}" ${build_dir}
	else
		CGO_ENABLED=${CGO_ENABLED} GOPROXY=${GOPROXY} GOOS=linux GOARCH=amd64 CC=x86_64-unknown-linux-gnu-gcc CXX=x86_64-unknown-linux-gnu-g++ go build -o "${AMD64_OUTPATH}" ${build_dir}
		CGO_ENABLED=${CGO_ENABLED} GOPROXY=${GOPROXY} GOOS=linux GOARCH=arm64 CC=aarch64-unknown-linux-gnu-gcc CXX=aarch64-unknown-linux-gnu-g++ go build -o "${ARM64_OUTPATH}" ${build_dir}
	fi
	# docker run --platform=linux/amd64 --rm -e CGO_ENABLED=${CGO_ENABLED} -e GOPROXY=${GOPROXY} -e GOOS=linux -e GOARCH=amd64 -v "${go_mod_cache}":/go/pkg/mod -v "$(pwd)":${WORK_DIR} -w ${WORK_DIR} ${build_image} go build -ldflags "${build_args}" -o "${AMD64_OUTPATH}" ${build_dir}
	# docker run --platform=linux/arm64 --rm -e CGO_ENABLED=${CGO_ENABLED} -e GOPROXY=${GOPROXY} -e GOOS=linux -e GOARCH=arm64 -v "${go_mod_cache}":/go/pkg/mod -v "$(pwd)":${WORK_DIR} -w ${WORK_DIR} ${build_image} go build -ldflags "${build_args}" -o "${ARM64_OUTPATH}" ${build_dir}
}

build::image() {
	local build_binary_dir="./_output/binary"
	local AMD64_OUTPATH="${build_binary_dir}/linux/amd64/${BASE_NAME}-$1"
	local ARM64_OUTPATH="${build_binary_dir}/linux/arm64/${BASE_NAME}-$1"
	local build_image_dir="./_output/image/$1/"
	local source_dir="./hack/contrib/docker/$1"
	local DOCKERFILE_BASE="Dockerfile.multiarch"
	mkdir -p "${build_image_dir}/amd64/" "${build_image_dir}/arm64/"
	chmod 777 "${build_image_dir}"
	if [ ! -f "${source_dir}/ignorebuild" ]; then
		if [ !${CACHE} ] || [ ! -f "${AMD64_OUTPATH}" ] || [ ! -f "${ARM64_OUTPATH}" ]; then
			build::binary "$1"
		fi
		cp "${AMD64_OUTPATH}" "${build_image_dir}/amd64/"
		cp "${ARM64_OUTPATH}" "${build_image_dir}/arm64/"
	fi
	cp -r ${source_dir}/* "${build_image_dir}"
	pushd "${build_image_dir}"
	echo "---> build image:$1"
	docker buildx use gobuilder || docker buildx create --use --name gobuilder
	# docker buildx build --platform linux/amd64,linux/arm64 --push --build-arg RELEASE_DESC="${release_desc}" --build-arg -t swr.cn-southwest-2.myhuaweicloud.com/wutong/wt-$1:${VERSION} -f "${DOCKERFILE_BASE}" .
	docker buildx build --platform linux/amd64,linux/arm64 --push --build-arg RELEASE_DESC="${release_desc}" -t wutongpaas/wt-$1:${VERSION} -f "${DOCKERFILE_BASE}" .
	# docker buildx rm gobuilder
	popd
	rm -rf "${build_image_dir}"

	if [ !${GITHUB_ACTIONS} ]; then
		rm -rf "${build_binary_dir}"
	fi
}

build::image::all() {
	for item in "${build_items[@]}"; do
		build::image "$item" "$1"
	done
}

build::binary::all() {
	for item in "${build_items[@]}"; do
		build::binary "$item" "$1"
	done
}

case $1 in
binary)
	if [ "$2" = "all" ]; then
		build::binary::all "$2"
	else
		build::binary "$2"
	fi
	;;
*)
	if [ "$1" = "all" ]; then
		build::image::all "$2"
	else
		build::image "$1" "$2"
	fi
	;;
esac
