#!/bin/bash
set -o errexit
WUTONG_REGISTRY=${WUTONG_REGISTRY:-'swr.cn-southwest-2.myhuaweicloud.com/wutong'}

base_items=(chaos eventlog gateway mesh-data-panel)

build::base() {
	local BASE_IMAGE_NAME="wt-$1-base:stable"
	pushd "$1"
	echo "---> build image:$BASE_IMAGE_NAME"
	docker buildx use mybuilder || docker buildx create --use --name mybuilder
	docker buildx build --platform linux/amd64,linux/arm64 --push -t $WUTONG_REGISTRY/$BASE_IMAGE_NAME .
	# docker buildx rm mybuilder
	popd
}

for item in "${base_items[@]}"; do
	build::base "$item"
done