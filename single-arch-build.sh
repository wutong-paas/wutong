#!/bin/bash
set -o errexit

# define package name
WORK_DIR=/go/src/github.com/wutong-paas/wutong
BASE_NAME=wutong
GOARCH=${BUILD_ARCH:-'amd64'}

GO_VERSION=1.17

GOPROXY=${GOPROXY:-'https://goproxy.io'}

if [ "$DISABLE_GOPROXY" == "true" ]; then
	GOPROXY=
fi
if [ -z "$GOOS" ]; then
	GOOS="linux"
fi
if [ "$DEBUG" ]; then
	set -x
fi
BRANCH=$(git symbolic-ref HEAD 2>/dev/null | cut -d"/" -f 3)
if [ -z "$VERSION" ]; then
	if [ -z "$TRAVIS_TAG" ]; then
		if [ -z "$TRAVIS_BRANCH" ]; then
			VERSION=$BRANCH-dev
		else
			VERSION=$TRAVIS_BRANCH-dev
		fi
	else
		VERSION=$TRAVIS_TAG
	fi
fi

buildTime=$(date +%F-%H)
git_commit=$(git log -n 1 --pretty --format=%h)

release_desc=${VERSION}-${git_commit}-${buildTime}
build_items=(api chaos gateway monitor mq webcli worker eventlog init-probe mesh-data-panel wtctl node resource-proxy)

build::binary() {
	echo "---> build binary:$1"
	home=$(pwd)
	local go_mod_cache="${home}/.cache"
	local OUTPATH="./_output/binary/linux/${BASE_NAME}-$1"
	local DOCKER_PATH="./hack/contrib/docker/$1"
	local build_image="golang:${GO_VERSION}"
	local build_args="-w -s -X github.com/wutong-paas/wutong/cmd.version=${release_desc}"
	local build_dir="./cmd/$1"
	local DOCKERFILE_BASE="Dockerfile"
	if [ -f "${DOCKER_PATH}/ignorebuild" ]; then
		return
	fi
	CGO_ENABLED=1
	if [ "$1" = "eventlog" ]; then
		if [ "$GOARCH" = "arm64" ]; then
			DOCKERFILE_BASE="Dockerfile.arm"
		fi
		docker build -t wutong.me/event-build:v1 -f "${DOCKER_PATH}/build/${DOCKERFILE_BASE}" "${DOCKER_PATH}/build/"
		build_image="wutong.me/event-build:v1"
	elif [ "$1" = "gateway" ]; then
		build_image="golang:${GO_VERSION}-alpine"
	elif [ "$1" = "monitor" ]; then
		CGO_ENABLED=0
	fi
	docker run --rm -e CGO_ENABLED=${CGO_ENABLED} -e GOPROXY=${GOPROXY} -e GOOS="${GOOS}" -v "${go_mod_cache}":/go/pkg/mod -v "$(pwd)":${WORK_DIR} -w ${WORK_DIR} ${build_image} go build -ldflags "${build_args}" -o "${OUTPATH}" ${build_dir}
}

build::image() {
	local build_binary_dir=./_output/binary/linux
	local OUTPATH="${build_binary_dir}/${BASE_NAME}-$1"
	local build_image_dir="./_output/image/$1/"
	local source_dir="./hack/contrib/docker/$1"
	local DOCKERFILE_BASE="Dockerfile"
	mkdir -p "${build_image_dir}" "${build_binary_dir}"
	chmod 777 "${build_image_dir}" "${build_binary_dir}"
	if [ ! -f "${source_dir}/ignorebuild" ]; then
		if [ ! ${CACHE} ] || [ ! -f "${OUTPATH}" ]; then
			build::binary "$1"
		fi
		cp "${OUTPATH}" "${build_image_dir}"
	fi
	cp -r ${source_dir}/* "${build_image_dir}"
	pushd "${build_image_dir}"
	echo "---> build image:$1"
	if [ "$GOARCH" = "arm64" ]; then
		if [ "$1" = "eventlog" ];then
			DOCKERFILE_BASE="Dockerfile.arm"
		elif [ "$1" = "mesh-data-panel" ];then
			DOCKERFILE_BASE="Dockerfile.arm"
		fi
	fi
	docker build --build-arg RELEASE_DESC="${release_desc}" --build-arg GOARCH="${GOARCH}" -t wt-$1:${VERSION} -f "${DOCKERFILE_BASE}" .

	if [ -f "${source_dir}/test.sh" ]; then
		"${source_dir}/test.sh" wt-$1:${VERSION}
	fi
	if [ "$2" = "push" ]; then
		docker tag wt-$1:${VERSION} swr.cn-southwest-2.myhuaweicloud.com/wutong/wt-$1:${VERSION}
		docker push swr.cn-southwest-2.myhuaweicloud.com/wutong/wt-$1:${VERSION}
	fi
	popd
	rm -rf "${build_image_dir}"
	rm -rf "${build_binary_dir}"
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