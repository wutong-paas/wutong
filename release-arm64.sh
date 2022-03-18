#!/bin/bash
set -o errexit

# define package name
WORK_DIR=/go/src/github.com/wutong-paas/wutong
BASE_NAME=wutong
IMAGE_BASE_NAME=${BUILD_IMAGE_BASE_NAME:-'wutongpaas'}
DOMESTIC_REGISTRY=${DOMESTIC_REGISTRY:-'swr.cn-southwest-2.myhuaweicloud.com'}
DOMESTIC_NAMESPACE=${DOMESTIC_REPO_NAME:-'wutong'}

GO_VERSION=1.13

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
# build_items=(api chaos gateway monitor mq webcli worker eventlog init-probe mesh-data-panel grctl node resource-proxy)
build_items=(eventlog)

build::binary() {
	echo "---> build binary:$1"
	home=$(pwd)
	local go_mod_cache="${home}/.cache"
	local OUTPATH="./_output/binary/$GOOS/${BASE_NAME}-$1"
	local DOCKER_PATH="./hack/contrib/docker/$1"
	local build_image="golang:${GO_VERSION}"
	local build_args="-w -s -X github.com/wutong-paas/wutong/cmd.version=${release_desc}"
	local build_dir="./cmd/$1"
	local build_tag=""
	local DOCKERFILE_BASE=${BUILD_DOCKERFILE_BASE:-'Dockerfile'}
	if [ -f "${DOCKER_PATH}/ignorebuild" ]; then
		return
	fi
	CGO_ENABLED=1
	# if [ "$1" = "eventlog" ]; then
	# 	if [ "$GOARCH" = "arm64" ]; then
	# 		DOCKERFILE_BASE="Dockerfile.arm"
	# 	fi
	    # docker buildx create --use --name wt-event || docker buildx rm wt-event
		# docker buildx build --push --platform linux/amd64 -t wutongpaas/event-build:v1 -f "${DOCKER_PATH}/build/${DOCKERFILE_BASE}" "${DOCKER_PATH}/build/"
		# docker buildx build --push --platform linux/arm64 -t wutongpaas/event-build:v1-arm64 -f "${DOCKER_PATH}/build/${DOCKERFILE_BASE}.arm" "${DOCKER_PATH}/build/"
		# docker buildx rm wt-event
		# build_image="wutongpaas/event-build:v1"
	# elif [ "$1" = "chaos" ]; then
	if [ "$1" = "chaos" ]; then
		build_dir="./cmd/builder"
	elif [ "$1" = "gateway" ]; then
		build_image="golang:1.13-alpine"
	elif [ "$1" = "monitor" ]; then
		CGO_ENABLED=0
	fi
	if [ "$1" = "eventlog" ]; then
		build_image="wutongpaas/event-build:v1"
		docker run --rm -e CGO_ENABLED=${CGO_ENABLED} -e GOPROXY=${GOPROXY} -e GOOS="${GOOS}" -v "${go_mod_cache}":/go/pkg/mod -v "$(pwd)":${WORK_DIR} -w ${WORK_DIR} ${build_image} go build -ldflags "${build_args}" -tags "${build_tag}" -o "${OUTPATH}" ${build_dir}

		build_image="wutongpaas/event-build:v1-arm64"
		docker run --platform=linux/arm64 --rm -e CGO_ENABLED=${CGO_ENABLED} -e GOPROXY=${GOPROXY} -e GOOS="${GOOS}" -v "${go_mod_cache}":/go/pkg/mod -v "$(pwd)":${WORK_DIR} -w ${WORK_DIR} ${build_image} go build -ldflags "${build_args}" -tags "${build_tag}" -o "${OUTPATH}-arm64" ${build_dir}
	else
		docker run --rm -e CGO_ENABLED=${CGO_ENABLED} -e GOPROXY=${GOPROXY} -e GOOS="${GOOS}" -v "${go_mod_cache}":/go/pkg/mod -v "$(pwd)":${WORK_DIR} -w ${WORK_DIR} ${build_image} go build -ldflags "${build_args}" -tags "${build_tag}" -o "${OUTPATH}" ${build_dir}
	fi
	if [ "$GOOS" = "windows" ]; then
		mv "$OUTPATH" "${OUTPATH}.exe"
	fi
}

build::image() {
	local OUTPATH="./_output/binary/$GOOS/${BASE_NAME}-$1"
	local build_image_dir="./_output/image/$1/"
	local source_dir="./hack/contrib/docker/$1"
	local BASE_IMAGE_VERSION=${BUILD_BASE_IMAGE_VERSION:-'3.15'}
	local DOCKERFILE_BASE=${BUILD_DOCKERFILE_BASE:-'Dockerfile'}
	mkdir -p "${build_image_dir}"
	chmod 777 "${build_image_dir}"
	if [ ! -f "${source_dir}/ignorebuild" ]; then
		if [ !${CACHE} ] || [ ! -f "${OUTPATH}" ]; then
			build::binary "$1"
		fi
		cp "${OUTPATH}" "${build_image_dir}"
	fi
	cp -r ${source_dir}/* "${build_image_dir}"
	pushd "${build_image_dir}"
	echo "---> build image:$1"
	# if [ "$1" = "eventlog" ];then
	# 	DOCKERFILE_BASE="Dockerfile.arm"
	# fi
	# if [ "$GOARCH" = "arm64" ]; then
	# 	if [ "$1" = "gateway" ]; then
	# 		BASE_IMAGE_VERSION="1.19.3.2-alpine"
	# 	elif [ "$1" = "eventlog" ];then
	# 		DOCKERFILE_BASE="Dockerfile.arm"
	# 	elif [ "$1" = "mesh-data-panel" ];then
	# 		DOCKERFILE_BASE="Dockerfile.arm"
	# 	fi
	# else
	# 	if [ "$1" = "gateway" ]; then
	# 		BASE_IMAGE_VERSION="1.19.3.2"
	# 	fi
	# fi
	if [ "$1" = "gateway" ]; then
		BASE_IMAGE_VERSION="1.19.3.2-alpine"
	fi
	if [ "$2" = "push" ]; then
		if [ $DOCKER_USERNAME ]; then
			docker login -u "$DOCKER_USERNAME" -p "$DOCKER_PASSWORD"
		fi
		
		if [ "${DOMESTIC_REGISTRY}" ]; then
			docker login -u "$DOMESTIC_DOCKER_USERNAME" -p "$DOMESTIC_DOCKER_PASSWORD" "${DOMESTIC_REGISTRY}"
		fi
	fi
	# Requirements:
	# 1. docker run --privileged --rm tonistiigi/binfmt --install all
	# 2. docker login dockerhub
	# 3. docker login huaweiyun-swr
	docker buildx create --use --name wt-builder || docker buildx rm wt-builder && docker buildx create --use --name wt-builder
	if [ "$1" = "eventlog" ]; then
		docker buildx build --push --platform linux/amd64 --build-arg RELEASE_DESC="${release_desc}" --build-arg BASE_IMAGE_VERSION="${BASE_IMAGE_VERSION}" -t "${IMAGE_BASE_NAME}/wt-$1:${VERSION}" -f "${DOCKERFILE_BASE}" .
		# docker buildx build --push --platform linux/amd64 --build-arg RELEASE_DESC="${release_desc}" --build-arg BASE_IMAGE_VERSION="${BASE_IMAGE_VERSION}" -t "${DOMESTIC_REGISTRY}/${DOMESTIC_NAMESPACE}/wt-$1:${VERSION}" -f "${DOCKERFILE_BASE}" .
		docker run --rm "${IMAGE_BASE_NAME}/wt-$1:${VERSION}" version

		docker buildx build --push --platform linux/arm64 --build-arg RELEASE_DESC="${release_desc}" --build-arg BASE_IMAGE_VERSION="${BASE_IMAGE_VERSION}" -t "${IMAGE_BASE_NAME}/wt-$1:${VERSION}-arm64" -f "${DOCKERFILE_BASE}.arm" .
		# docker buildx build --push --platform linux/arm64 --build-arg RELEASE_DESC="${release_desc}" --build-arg BASE_IMAGE_VERSION="${BASE_IMAGE_VERSION}" -t "${DOMESTIC_REGISTRY}/${DOMESTIC_NAMESPACE}/wt-$1:${VERSION}" -f "${DOCKERFILE_BASE}.arm" .
		docker run --platform linux/arm64 --rm "${IMAGE_BASE_NAME}/wt-$1:${VERSION}-arm64" version
	else
		docker buildx build --push --platform linux/amd64,linux/arm64 --build-arg RELEASE_DESC="${release_desc}" --build-arg BASE_IMAGE_VERSION="${BASE_IMAGE_VERSION}" -t "${IMAGE_BASE_NAME}/wt-$1:${VERSION}" -f "${DOCKERFILE_BASE}" .
		docker buildx build --push --platform linux/amd64,linux/arm64 --build-arg RELEASE_DESC="${release_desc}" --build-arg BASE_IMAGE_VERSION="${BASE_IMAGE_VERSION}" -t "${DOMESTIC_REGISTRY}/${DOMESTIC_NAMESPACE}/wt-$1:${VERSION}" -f "${DOCKERFILE_BASE}" .
		docker run --rm "${IMAGE_BASE_NAME}/wt-$1:${VERSION}" version
	fi
	docker buildx rm wt-builder
	if [ $? -ne 0 ]; then
		echo "image version is different ${release_desc}"
		exit 1
	fi
	if [ -f "${source_dir}/test.sh" ]; then
		"${source_dir}/test.sh" "${IMAGE_BASE_NAME}/wt-$1:${VERSION}"
	fi
	# if [ "$2" = "push" ]; then
	# 	if [ $DOCKER_USERNAME ]; then
	# 		docker login -u "$DOCKER_USERNAME" -p "$DOCKER_PASSWORD"
	# 		docker push "${IMAGE_BASE_NAME}/wt-$1:${VERSION}"
	# 	fi
		
	# 	if [ "${DOMESTIC_REGISTRY}" ]; then
	# 		docker tag "${IMAGE_BASE_NAME}/wt-$1:${VERSION}" "${DOMESTIC_REGISTRY}/${DOMESTIC_NAMESPACE}/wt-$1:${VERSION}"
	# 		docker login -u "$DOMESTIC_DOCKER_USERNAME" -p "$DOMESTIC_DOCKER_PASSWORD" "${DOMESTIC_REGISTRY}"
	# 		docker push "${DOMESTIC_REGISTRY}/${DOMESTIC_NAMESPACE}/wt-$1:${VERSION}"
	# 	fi
	# fi
	popd
	rm -rf "${build_image_dir}"
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
