#!/bin/bash
set -o errexit

# define package name
IMAGE_REPO=${IMAGE_REPO:-'wutongpaas'}

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
build_items=(api chaos gateway monitor mq webcli worker eventlog init-probe mesh-data-panel grctl node resource-proxy)
# build_items=(api chaos gateway mq webcli worker init-probe mesh-data-panel grctl node resource-proxy)
# build_items=(eventlog)

build::all() {
	echo "---> build image: wt-all:${VERSION}"
	# docker buildx create --use --name wt-all-builder
	docker buildx use wt-all-builder
	docker buildx build --push --platform linux/amd64 -t ${IMAGE_REPO}/wt-all:${VERSION} -f "./hack/contrib/docker/all/Dockerfile.multiarch" .
	# docker buildx rm wt-all-builder
}

build::image() {
	local build_image_dir="./_output/image/$1/"
	local source_dir="./hack/contrib/docker/$1"
	local BASE_IMAGE_VERSION=${BUILD_BASE_IMAGE_VERSION:-'3.15'}
	local DOCKERFILE_BASE=${BUILD_DOCKERFILE_BASE:-'Dockerfile.multiarch'}
	mkdir -p "${build_image_dir}"
	chmod 777 "${build_image_dir}"

	cp -r ${source_dir}/* "${build_image_dir}"
	pushd "${build_image_dir}"
	echo "---> build image: $1"

	# docker buildx create --use --name $1-builder
	docker buildx use $1-builder
	# if [ "$1" = "eventlog" ]; then
	# 	docker buildx build --push --platform linux/amd64 --build-arg IMAGE_REPO="${IMAGE_REPO}" --build-arg RELEASE_DESC="${release_desc}" --build-arg BASE_IMAGE_VERSION="${BASE_IMAGE_VERSION}" --build-arg VERSION="${VERSION}" -t "${IMAGE_REPO}/wt-$1:${VERSION}" .
	# 	docker buildx build --push --platform linux/arm64 --build-arg IMAGE_REPO="${IMAGE_REPO}" --build-arg RELEASE_DESC="${release_desc}" --build-arg BASE_IMAGE_VERSION="${BASE_IMAGE_VERSION}" --build-arg VERSION="${VERSION}" -t "${IMAGE_REPO}/wt-$1:${VERSION}-arm64" -f Dockerfile.arm .
	# else
	# 	docker buildx build --push --platform linux/amd64,linux/arm64 --build-arg IMAGE_REPO="${IMAGE_REPO}" --build-arg RELEASE_DESC="${release_desc}" --build-arg BASE_IMAGE_VERSION="${BASE_IMAGE_VERSION}" --build-arg VERSION="${VERSION}" -t "${IMAGE_REPO}/wt-$1:${VERSION}" -f "${DOCKERFILE_BASE}" .
	# fi
	docker buildx build --push --platform linux/amd64 --build-arg IMAGE_REPO="${IMAGE_REPO}" --build-arg RELEASE_DESC="${release_desc}" --build-arg BASE_IMAGE_VERSION="${BASE_IMAGE_VERSION}" --build-arg VERSION="${VERSION}" -t "${IMAGE_REPO}/wt-$1:${VERSION}" -f "${DOCKERFILE_BASE}" .
	
	# docker buildx rm $1-builder
	docker pull --platform linux/amd64 "${IMAGE_REPO}/wt-$1:${VERSION}"
	docker run --platform linux/amd64 --rm "${IMAGE_REPO}/wt-$1:${VERSION}" version
	if [ $? -ne 0 ]; then
		echo "image version is different ${release_desc}"
		exit 1
	fi
	if [ -f "${source_dir}/test.sh" ]; then
		"${source_dir}/test.sh" "${IMAGE_REPO}/wt-$1:${VERSION}"
	fi
	popd
	rm -rf "${build_image_dir}"
}

build::image::all() {
	for item in "${build_items[@]}"; do
		build::image "$item"
	done
}

build::all
if [ "$1" = "all" ]; then
	build::image::all
else
	build::image "$1"
fi